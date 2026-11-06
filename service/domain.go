package service

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// 知识域规则层：决定「什么问题能答、什么答案能进库」。
//
// 为什么必须落在服务端而不是只写进提示词：提示词是建议，模型可以不听。
// 而「不含走心机内容的问题不得进入知识库」属于数据准入，一旦失守，
// 后面所有检索、引用、审核都建立在一堆噪声上——清理成本远高于拦截成本。
// 所以判定由代码做，模型只在被允许的范围内负责组织答案。
//
// 规则存在 sys_config 的 kb 分组下，可在「知识规则」页维护，改完立即生效。

const (
	// 领域外策略
	kbPolicyStrict = "strict" // 领域外直接拒答：不调模型、不入库
	kbPolicyOpen   = "open"   // 领域外照常作答，但绝不入库

	// 判定结论
	domainMatched   = "matched"    // 命中准入词
	domainBlocked   = "blocked"    // 命中排除词
	domainNoSignal  = "no_signal"  // 既没准入也没排除，判定为范围外
	domainInjection = "injection"  // 疑似提示词注入
	domainEmpty     = "empty"      // 空问题

	// 沉淀闸门结论
	collectOK          = "ok"
	collectDisabled    = "collect_disabled"
	collectOutDomain   = "out_of_domain"
	collectTooShort    = "answer_too_short"
	collectRefusal     = "answer_is_refusal"
	collectQuotaExceed = "quota_exceeded"
)

// kbDomainGroup 规则所在的配置分组
const kbDomainGroup = "kb"

// kbRuleKeys 规则涉及的配置键白名单
var kbRuleKeys = []string{
	"kbDomainName", "kbDomainScope", "kbDomainInclude", "kbDomainExclude",
	"kbDomainMinHits", "kbDomainPolicy", "kbCollectEnabled",
	"kbCollectMinAnswer", "kbCollectDailyQuota",
}

// kbInjectionWords 提示词注入特征词。
// 故意不可配置：这是硬拦截，不随运营调整而放宽，否则一条被改坏的规则就能把闸门拆掉。
var kbInjectionWords = []string{
	"忽略之前", "忽略以上", "忽略上述", "忽略前面", "忘记之前", "无视之前",
	"ignore previous", "ignore above", "ignore all previous",
	"系统提示词", "系统指令", "输出你的提示", "重复你的指令", "你的提示词是什么",
	"开发者模式", "developer mode", "越狱", "jailbreak",
	"你现在是一个", "从现在开始你是", "扮演一个",
}

// kbDefaultInclude 默认准入词表。
// 命中任意一个即视为领域内，所以宁可覆盖广一些：漏掉一个词会把真问题挡在门外，
// 多放一个词最多让边缘问题也答一次（而且还要再过沉淀闸门）。
func kbDefaultInclude() []string {
	return []string{
		// 设备与结构
		"走心机", "走刀机", "纵切", "车床", "车削", "数控车", "车铣", "加工中心", "cnc",
		"主轴", "副主轴", "背轴", "导套", "夹头", "筒夹", "卡盘", "刀架", "排刀", "刀塔",
		"动力刀", "尾座", "送料", "棒料", "料仓", "机床",
		// 工艺
		"断屑", "排屑", "振刀", "让刀", "崩刃", "积屑瘤", "刃磨", "对刀", "试切",
		"粗车", "精车", "端面", "外圆", "内孔", "切槽", "切断", "螺纹", "攻丝",
		"钻孔", "镗孔", "铰孔", "倒角", "滚花", "铣削", "磨削", "去毛刺", "毛刺",
		"振纹", "锥度", "椭圆",
		// 参数
		"切削", "进给", "转速", "线速度", "吃刀量", "背吃刀量", "切削液", "冷却液",
		"表面粗糙度", "粗糙度", "光洁度", "公差", "尺寸精度", "圆度", "同轴度",
		"跳动", "热变形", "余量", "留量", "切削速度",
		// 刀具
		"刀具", "车刀", "铣刀", "钻头", "丝锥", "刀片", "刀杆", "刀柄", "硬质合金",
		"涂层", "刀尖圆弧", "刀补", "可转位", "刀尖", "刀具寿命", "磨损",
		// 编程
		"g代码", "m代码", "宏程序", "子程序", "程序段", "坐标系", "刀尖半径补偿",
		"仿真", "cam", "宏变量", "参数编程",
		// 材料
		"不锈钢", "铝合金", "钛合金", "黄铜", "碳钢", "合金钢", "易切削钢", "铸铁",
		"pom", "尼龙", "淬火",
		// 质量与检测
		"首件", "巡检", "量具", "千分尺", "卡尺", "粗糙度仪", "废品", "返修", "精度",
		// 维护与管理
		"点检", "保养", "润滑", "报警", "伺服", "丝杠", "导轨", "轴承", "夹具",
		"工艺卡", "作业指导书", "sop", "故障", "维修", "装夹", "校正",
	}
}

// kbDefaultExclude 默认排除词表。
// 只放「明确无关」的词，ambiguous 的词不要放——宁可让沉淀闸门在后面兜住，
// 也不要在入口把真问题误伤掉。
func kbDefaultExclude() []string {
	return []string{
		// 生活服务
		"天气", "气温", "股票", "基金", "彩票", "房价", "快递", "外卖", "旅游",
		"机票", "电影", "电视剧", "游戏", "王者荣耀", "原神", "笑话", "星座",
		"算命", "恋爱", "情感", "作文", "简历", "面试", "翻译",
		// 其他技术栈
		"python", "javascript", "typescript", "java", "php", "html", "css",
		"react", "vue", "sql", "爬虫", "黑客", "挖矿", "区块链", "比特币",
		"虚拟币", "docker", "linux",
		// 其它领域
		"政治", "选举", "宗教",
	}
}

// kbDefaultScope 默认领域范围描述，会注入兜底作答的提示词
const kbDefaultScope = "只收走心机（纵切自动车床）、数控车削、加工中心相关的资料与问答：" +
	"设备结构与调整、刀具与磨损、切削参数、编程指令、装夹与对刀、材料与断屑排屑、" +
	"质量检测、保养与故障处理。不收录与机加工无关的内容。"

// kbRule 知识域规则
type kbRule struct {
	Name       string   `json:"name"`
	Scope      string   `json:"scope"`
	Include    []string `json:"include"`
	Exclude    []string `json:"exclude"`
	MinHits    int      `json:"minHits"`
	Policy     string   `json:"policy"`
	Collect    bool     `json:"collectEnabled"`
	MinAnswer  int      `json:"minAnswer"`
	DailyQuota int      `json:"dailyQuota"`
}

// kbDefaultRule 出厂规则
func kbDefaultRule() kbRule {
	return kbRule{
		Name:       "走心机与数控加工",
		Scope:      kbDefaultScope,
		Include:    kbDefaultInclude(),
		Exclude:    kbDefaultExclude(),
		MinHits:    1,
		Policy:     kbPolicyStrict,
		Collect:    true,
		MinAnswer:  20,
		DailyQuota: 50,
	}
}

// loadKbRule 读取规则，未配置的项用默认值兜底
func loadKbRule() kbRule {
	rule := kbDefaultRule()
	values, _ := dao.ConfigMap()
	pick := func(key string) (string, bool) {
		v, ok := values[key]
		v = strings.TrimSpace(v)
		if !ok || v == "" {
			return "", false
		}
		return v, true
	}

	if v, ok := pick("kbDomainName"); ok {
		rule.Name = v
	}
	if v, ok := pick("kbDomainScope"); ok {
		rule.Scope = v
	}
	if v, ok := pick("kbDomainInclude"); ok {
		rule.Include = splitWordList(v)
	}
	if v, ok := pick("kbDomainExclude"); ok {
		rule.Exclude = splitWordList(v)
	}
	if v, ok := pick("kbDomainMinHits"); ok {
		rule.MinHits = int(clampFloat(parseFloat(v, float64(rule.MinHits)), 1, 5))
	}
	if v, ok := pick("kbDomainPolicy"); ok && (v == kbPolicyStrict || v == kbPolicyOpen) {
		rule.Policy = v
	}
	if v, ok := pick("kbCollectEnabled"); ok {
		rule.Collect = v != "false"
	}
	if v, ok := pick("kbCollectMinAnswer"); ok {
		rule.MinAnswer = int(clampFloat(parseFloat(v, float64(rule.MinAnswer)), 5, 500))
	}
	if v, ok := pick("kbCollectDailyQuota"); ok {
		rule.DailyQuota = int(clampFloat(parseFloat(v, float64(rule.DailyQuota)), 0, 1000))
	}
	if len(rule.Include) == 0 {
		rule.Include = kbDefaultInclude()
	}
	return rule
}

// splitWordList 把配置里的词表还原成切片，支持换行、逗号、顿号、分号分隔
func splitWordList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '\n', '\r', ',', '，', '、', ';', '；', '|':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]bool, len(fields))
	for _, item := range fields {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

// ---------- 判定 ----------

// domainVerdict 一次领域判定结论
type domainVerdict struct {
	In      bool     `json:"inDomain"`
	Reason  string   `json:"reason"`
	Hits    []string `json:"hits"`
	Blocked []string `json:"blocked"`
}

// matchWords 在原文与去空格副本里同时找词。
// 去空格是为了覆盖「G 代码」「CNC 车床」这类中间带空格的写法。
func matchWords(text, compact string, words []string, limit int) []string {
	hits := make([]string, 0, 8)
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}
		if strings.Contains(text, word) || strings.Contains(compact, word) {
			hits = append(hits, word)
			if limit > 0 && len(hits) >= limit {
				break
			}
		}
	}
	return hits
}

// judge 判定一个问题是否属于本知识域。
// 顺序有讲究：
//  1. 注入特征优先拦截——带着「走心机」三个字的注入请求也是注入；
//  2. 命中准入词即放行——即使同时出现「翻译」这种排除词，
//     也不能挡住「翻译一下这段 G 代码」这种真问题；
//  3. 都没有才看排除词，最后落到「无信号」。
func (rule kbRule) judge(question string) domainVerdict {
	text := strings.ToLower(strings.TrimSpace(question))
	compact := strings.Join(strings.Fields(text), "")
	if compact == "" {
		return domainVerdict{Reason: domainEmpty, Hits: []string{}, Blocked: []string{}}
	}
	if hit := matchWords(text, compact, kbInjectionWords, 1); len(hit) > 0 {
		return domainVerdict{Reason: domainInjection, Hits: []string{}, Blocked: hit}
	}

	include := matchWords(text, compact, rule.Include, 8)
	minHits := rule.MinHits
	if minHits < 1 {
		minHits = 1
	}
	if len(include) >= minHits {
		return domainVerdict{In: true, Reason: domainMatched, Hits: include, Blocked: []string{}}
	}
	if blocked := matchWords(text, compact, rule.Exclude, 5); len(blocked) > 0 {
		return domainVerdict{Reason: domainBlocked, Hits: include, Blocked: blocked}
	}
	return domainVerdict{Reason: domainNoSignal, Hits: include, Blocked: []string{}}
}

// explain 把判定结论翻译成一句人话，给页面和拒答文案复用
func (v domainVerdict) explain(rule kbRule) string {
	switch v.Reason {
	case domainMatched:
		return "命中领域词：" + strings.Join(v.Hits, "、")
	case domainBlocked:
		return "命中范围外词汇：" + strings.Join(v.Blocked, "、")
	case domainInjection:
		return "疑似尝试改写助手指令，已拦截"
	case domainEmpty:
		return "问题为空"
	default:
		return "没有命中任何「" + rule.Name + "」相关词汇"
	}
}

// ---------- 沉淀闸门 ----------

// collectDecision 一条问答对能否入库的结论
type collectDecision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// kbRefusalPrefixes 模型拒答的典型开头，这类答案没有沉淀价值
var kbRefusalPrefixes = []string{
	"抱歉", "很抱歉", "对不起", "我无法", "我不能", "我没办法",
	"无法回答", "作为一个", "作为 ai", "作为ai",
}

// looksLikeRefusal 判断答案本身是不是一段拒答
func looksLikeRefusal(answer string) bool {
	runes := []rune(strings.TrimSpace(answer))
	if len(runes) > 24 {
		runes = runes[:24]
	}
	head := strings.ToLower(string(runes))
	for _, prefix := range kbRefusalPrefixes {
		if strings.Contains(head, prefix) {
			return true
		}
	}
	return false
}

// gate 沉淀闸门，所有入库路径的必经之处。
//
// 分来源判断：带出处的答案来自「已发布资料」，本身就是领域内内容，只校验质量与配额；
// 无出处的兜底答案必须重新过一遍领域判定——它是唯一可能把噪声带进库的入口。
func (rule kbRule) gate(question, answer, source string) collectDecision {
	if !rule.Collect {
		return collectDecision{Reason: collectDisabled, Message: "自动沉淀已关闭，只有人工录入能进库"}
	}
	answer = strings.TrimSpace(answer)
	if len([]rune(answer)) < rule.MinAnswer {
		return collectDecision{
			Reason:  collectTooShort,
			Message: "答案不足 " + strconv.Itoa(rule.MinAnswer) + " 字，入库后没有复用价值",
		}
	}
	if looksLikeRefusal(answer) {
		return collectDecision{Reason: collectRefusal, Message: "答案本身是拒答或不确定，不入库"}
	}
	if source == model.QaSourceFallback {
		if verdict := rule.judge(question); !verdict.In {
			return collectDecision{Reason: collectOutDomain, Message: verdict.explain(rule)}
		}
	}
	if rule.DailyQuota > 0 {
		if n, err := dao.CountKbQaCreatedSince(dayStart(time.Now())); err == nil && n >= int64(rule.DailyQuota) {
			return collectDecision{
				Reason:  collectQuotaExceed,
				Message: "今日自动入库已达上限 " + strconv.Itoa(rule.DailyQuota) + " 条，明日恢复",
			}
		}
	}
	return collectDecision{Allowed: true, Reason: collectOK}
}

// dayStart 当天零点，用于每日配额统计
func dayStart(now time.Time) time.Time {
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, now.Location())
}

// ---------- 接口 ----------

// kbRuleView 读取规则、出厂默认值与沉淀概况
func kbRuleView(c *adminCtx, r *http.Request) (interface{}, error) {
	rule := loadKbRule()
	statusCounts, _ := dao.CountKbQaByStatus()
	fallbackPending, _ := dao.CountKbQaBySourceAndStatus(model.QaSourceFallback, model.QaCandidate)
	todayCollected, _ := dao.CountKbQaCreatedSince(dayStart(time.Now()))

	return map[string]interface{}{
		"rule":      rule,
		"defaults":  kbDefaultRule(),
		"injection": kbInjectionWords,
		"stats": map[string]interface{}{
			"todayCollected":  todayCollected,
			"pending":         statusCounts[model.QaCandidate],
			"published":       statusCounts[model.QaPublished],
			"fallbackPending": fallbackPending,
			"includeCount":    len(rule.Include),
			"excludeCount":    len(rule.Exclude),
		},
	}, nil
}

// kbRuleSave 保存规则。词表逐项校验，避免一个空词吃满整个匹配
func kbRuleSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Name       string   `json:"name"`
		Scope      string   `json:"scope"`
		Include    []string `json:"include"`
		Exclude    []string `json:"exclude"`
		MinHits    int      `json:"minHits"`
		Policy     string   `json:"policy"`
		Collect    *bool    `json:"collectEnabled"`
		MinAnswer  int      `json:"minAnswer"`
		DailyQuota int      `json:"dailyQuota"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "走心机与数控加工"
	}
	if len([]rune(name)) > 32 {
		return nil, badRequest("领域名称不能超过 32 个字")
	}
	scope := strings.TrimSpace(req.Scope)
	if len([]rune(scope)) > 1000 {
		return nil, badRequest("领域范围描述不能超过 1000 个字")
	}
	if scope == "" {
		scope = kbDefaultScope
	}

	include := normalizeWordList(req.Include, 300)
	if len(include) == 0 {
		return nil, badRequest("准入词表不能为空，否则任何人都进不了知识库")
	}
	exclude := normalizeWordList(req.Exclude, 300)

	if req.MinHits < 1 || req.MinHits > 5 {
		return nil, badRequest("最少命中词数需在 1 - 5 之间")
	}
	policy := strings.TrimSpace(req.Policy)
	if policy != kbPolicyStrict && policy != kbPolicyOpen {
		return nil, badRequest("范围外策略取值不合法")
	}
	if req.MinAnswer < 5 || req.MinAnswer > 500 {
		return nil, badRequest("最短答案长度需在 5 - 500 之间")
	}
	if req.DailyQuota < 0 || req.DailyQuota > 1000 {
		return nil, badRequest("每日入库上限需在 0 - 1000 之间")
	}

	collect := "true"
	if req.Collect != nil && !*req.Collect {
		collect = "false"
	}

	items := map[string]string{
		"kbDomainName":        name,
		"kbDomainScope":       scope,
		"kbDomainInclude":     strings.Join(include, "\n"),
		"kbDomainExclude":     strings.Join(exclude, "\n"),
		"kbDomainMinHits":     strconv.Itoa(req.MinHits),
		"kbDomainPolicy":      policy,
		"kbCollectEnabled":    collect,
		"kbCollectMinAnswer":  strconv.Itoa(req.MinAnswer),
		"kbCollectDailyQuota": strconv.Itoa(req.DailyQuota),
	}
	for _, key := range kbRuleKeys {
		value, ok := items[key]
		if !ok {
			continue
		}
		if err := dao.SaveConfig(key, value); err != nil {
			return nil, serverError("保存 " + key + " 失败: " + err.Error())
		}
	}
	return map[string]interface{}{"rule": loadKbRule()}, nil
}

// normalizeWordList 词表规整：去空格、去重、丢掉空词与超长词，并限制总量
func normalizeWordList(raw []string, limit int) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] || len([]rune(item)) > 20 {
			continue
		}
		seen[item] = true
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// kbRuleTest 规则试跑：输入一句话，看它会不会被判定为领域内、会不会被沉淀。
// 这是让规则可被验证的关键——管理员改完词表能立刻看到效果，而不是等线上出问题。
func kbRuleTest(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Question string `json:"question"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return nil, badRequest("请输入要试跑的问题")
	}
	if len([]rune(question)) > 500 {
		return nil, badRequest("问题太长，请控制在 500 字以内")
	}

	rule := loadKbRule()
	verdict := rule.judge(question)
	// 用一个长度达标的占位答案，只为验证闸门的前置条件，
	// 真实答案长度要到作答完成后才能确定。
	placeholder := strings.Repeat("测", rule.MinAnswer)
	gate := rule.gate(question, placeholder, model.QaSourceFallback)

	explain := verdict.explain(rule)
	next := ""
	switch {
	case !verdict.In && rule.Policy == kbPolicyStrict:
		next = "会被直接拒答：不调用模型，不写入知识库。"
	case !verdict.In:
		next = "会照常作答，但答案不会入库。"
	case !gate.Allowed:
		next = "会作答，但这次不满足入库条件：" + gate.Message
	default:
		next = "会作答，并沉淀为待审核问答（无出处，需人工核实后才能定稿）。"
	}

	return map[string]interface{}{
		"question":     question,
		"verdict":      verdict,
		"explain":      explain,
		"gate":         gate,
		"wouldCollect": gate.Allowed,
		"policy":       rule.Policy,
		"next":         next,
	}, nil
}
