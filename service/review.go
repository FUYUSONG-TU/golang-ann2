package service

import (
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// 自动审核：把「待审核」这一列的积压交给规则层自己处理。
//
// 审核要回答的其实只有三个问题——这条问题是不是本领域的、答案有没有实质内容、
// 有没有可追溯的依据。三件事都能用规则判定，不必每次都靠人眼。
// 人真正该看的是规则拿不准的那部分，而不是逐条点「定稿」。
//
// 反过来，自动审核的边界也必须清楚：它只能放行「依据充分」或「高置信」的答案。
// 机加工场景里，一段编出来的切削参数会被当成标准执行，直接进车间。
// 所以默认配置下，无出处的兜底答案一律不进自动定稿，只标分数交给人。

const (
	// 审核结论
	reviewPass   = "pass"   // 可以定稿
	reviewHold   = "hold"   // 拿不准，留给人工
	reviewReject = "reject" // 明确不合格，直接驳回
)

// kbAbsoluteWords 绝对化表述。
// 这类话术是模型在「答案其实没有依据」时的典型伪装，也是学员最容易照做的一类。
// 命中不驳回但一定压分，压到不够线自然就留给人工了。
var kbAbsoluteWords = []string{
	"绝对", "百分之百", "百分百", "100%", "一定能", "保证不会", "永远不会",
	"零风险", "肯定不会", "万无一失", "包解决",
}

// kbSelfRefWords 模型自我描述。答案里出现这些，说明这段文字是对话而不是知识。
var kbSelfRefWords = []string{
	"作为一个ai", "作为ai", "ai语言模型", "语言模型", "我无法访问", "我的知识截止", "作为人工智能",
}

// kbHedgeWords 含糊表述。轻微压分：技术回答里出现「可能」很正常，
// 但如果通篇没有一句确定的结论，这条答案就没有复用价值。
var kbHedgeWords = []string{"不太清楚", "说不好", "不确定", "因人而异", "仅供参考"}

// kbParamPattern 具体参数特征：带单位的数值，或者 G/M 代码。
// 这是「可执行」的最强信号——「进给 0.08 mm/r」比「适当降低进给」有用得多。
var kbParamPattern = regexp.MustCompile(`(?i)(\d+(\.\d+)?\s*(mm/r|mm|r/min|m/min|rpm|μm|um|°|度|丝|mpa|bar|n·m)|\b[gm]\d{2,3}\b)`)

// kbStepPattern 结构化特征：分点、编号或分步描述
var kbStepPattern = regexp.MustCompile(`(1[\.、]|①|第一步|首先.{0,20}然后|\n\s*\n)`)

// reviewResult 一次自动审核的结论
type reviewResult struct {
	Verdict string `json:"verdict"`
	Score   int    `json:"score"`
	Note    string `json:"note"`
}

// reviewCtx 一次审核所需的上下文。
// published 保存归一化后的已定稿问题文本，用于识别重复；为 nil 时跳过重复检查
// （沉淀后的即时审核走这条路，避免每问一句就把整张定稿表拉一遍）。
type reviewCtx struct {
	rule      kbRule
	published map[string]string // 归一化问题 -> 原问题
}

// 定稿问题集缓存。审核台每次翻页、每次预判都要用它查重，
// 每次都把整张定稿表拉一遍没有必要；30 秒的窗口足够挡住重复查询，
// 又短到不会让「刚定稿的条目」在页面上迟迟不生效。
var (
	publishedQaMu    sync.Mutex
	publishedQaCache []string
	publishedQaAt    time.Time
)

const publishedQaTTL = 30 * time.Second

// publishedQaQuestions 已定稿问题文本，带短 TTL 缓存
func publishedQaQuestions() []string {
	publishedQaMu.Lock()
	defer publishedQaMu.Unlock()
	if publishedQaCache != nil && time.Since(publishedQaAt) < publishedQaTTL {
		return publishedQaCache
	}
	list, err := dao.ListKbQaPublishedQuestions(8000)
	if err != nil {
		log.Printf("自动审核加载已定稿问题失败: %v", err)
		return publishedQaCache
	}
	publishedQaCache = list
	publishedQaAt = time.Now()
	return list
}

// invalidatePublishedQa 定稿状态发生变化后主动失效缓存，
// 否则「刚定稿完再点一次自动审核」会看到上一次的旧查重结果。
func invalidatePublishedQa() {
	publishedQaMu.Lock()
	publishedQaAt = time.Time{}
	publishedQaMu.Unlock()
}

// newReviewCtx 构造审核上下文。loadPublished 为 true 时带上已定稿问题用于查重。
func newReviewCtx(rule kbRule, loadPublished bool) reviewCtx {
	ctx := reviewCtx{rule: rule}
	if !loadPublished {
		return ctx
	}
	questions := publishedQaQuestions()
	ctx.published = make(map[string]string, len(questions))
	for _, q := range questions {
		if key := normalizeQaQuestion(q); key != "" {
			ctx.published[key] = q
		}
	}
	return ctx
}

// normalizeQaQuestion 归一化问题文本：去掉标点与空格、统一小写。
// 只用来判重，不参与相似度计算——「G74 怎么切槽？」和「g74怎么切槽」是同一个问题。
func normalizeQaQuestion(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(raw) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// previewReview 只算不写，用于页面上提前显示「这条会被怎么判」。
// 走和正式审核完全相同的 reviewCtx，所以预判结论与点按钮后的结果一致。
func previewReview(ctx reviewCtx, item model.KbQa) reviewResult {
	return ctx.review(item)
}

// review 按规则层给一条候选打分并给出结论。
//
// 顺序：先做硬性驳回（不合格就是不合格，不必算分），再按证据打分。
// 硬性条件全部来自「数据准入」，与领域判定共用同一套词表，
// 所以改词表时不需要再维护第二份规则。
func (ctx reviewCtx) review(item model.KbQa) reviewResult {
	rule := ctx.rule
	if !rule.AutoReview {
		return reviewResult{Verdict: reviewHold, Note: "自动审核已关闭，留待人工"}
	}

	answer := strings.TrimSpace(item.Answer)
	answerRunes := len([]rune(answer))
	refs := len(parseQaRefs(item.Refs))

	// ---- 硬性驳回 ----
	if answerRunes < rule.MinAnswer {
		return reviewResult{
			Verdict: reviewReject, Score: 0,
			Note: "自动驳回：答案不足 " + strconv.Itoa(rule.MinAnswer) + " 字，入库后没有复用价值",
		}
	}
	if looksLikeRefusal(answer) {
		return reviewResult{Verdict: reviewReject, Score: 0, Note: "自动驳回：答案本身是拒答或免责表述"}
	}
	if hit := matchWords(answer, strings.Join(strings.Fields(answer), ""), kbSelfRefWords, 1); len(hit) > 0 {
		return reviewResult{Verdict: reviewReject, Score: 0, Note: "自动驳回：答案里出现模型自我描述（" + hit[0] + "），不是知识内容"}
	}
	if refs == 0 && !ctx.rule.judge(item.Question).In {
		return reviewResult{
			Verdict: reviewReject, Score: 0,
			Note: "自动驳回：" + ctx.rule.judge(item.Question).explain(ctx.rule),
		}
	}
	if ctx.published != nil {
		if key := normalizeQaQuestion(item.Question); key != "" {
			if exists, ok := ctx.published[key]; ok {
				return reviewResult{
					Verdict: reviewReject, Score: 0,
					Note: "自动驳回：已存在相同问题的定稿条目「" + truncate(exists, 40) + "」，如内容更好请恢复后合并",
				}
			}
		}
	}

	// ---- 证据分 ----
	// 起步 55：一条能进候选池的答案本身已经过了领域与质量闸门，
	// 分数只用来区分「可以直接用」和「还得人看一眼」。
	score := 55
	good := make([]string, 0, 6)
	bad := make([]string, 0, 3)

	if refs > 0 {
		score += 18
		good = append(good, "有资料出处")
	}
	if answerRunes >= rule.AutoMinAnswer {
		score += 7
	}
	if answerRunes >= 120 {
		score += 3
	}
	lower := strings.ToLower(answer)
	compact := strings.Join(strings.Fields(lower), "")
	if kbParamPattern.MatchString(answer) {
		score += 8
		good = append(good, "含具体参数")
	}
	if kbStepPattern.MatchString(answer) {
		score += 4
	}
	if item.AskCount >= 2 {
		score += 3
	}
	if item.AskCount >= 3 {
		score += 2
	}
	if item.UpCount > 0 {
		score += 3
		good = append(good, "有学员好评")
	}
	if hits := ctx.rule.judge(item.Question).Hits; len(hits) >= 2 {
		score += 2
	}

	risky := false
	if hit := matchWords(lower, compact, kbAbsoluteWords, 2); len(hit) > 0 {
		score -= 22
		risky = true
		bad = append(bad, "绝对化表述（"+strings.Join(hit, "、")+"）")
	}
	if hit := matchWords(lower, compact, kbHedgeWords, 2); len(hit) > 0 {
		score -= 8
		bad = append(bad, "结论含糊（"+strings.Join(hit, "、")+"）")
	}
	if item.DownCount > 0 {
		score -= 30
		risky = true
		bad = append(bad, "收到过差评")
	}
	if answerRunes < rule.AutoMinAnswer {
		bad = append(bad, "答案偏短")
	}
	if refs == 0 {
		bad = append(bad, "没有资料来源")
	}

	score = clampScore(score)

	// 自动定稿的三道门：分数够、没有风险标记、出处合规。
	// 三道门任一不过就留给人——自动审核的价值在于减少无效点击，
	// 不在于把所有东西都自动放过去。
	if answerRunes >= rule.AutoMinAnswer && !risky && score >= rule.AutoScore &&
		(refs > 0 || rule.AutoAllowUnsourced) {
		prefix := "自动定稿："
		if refs == 0 {
			// 无出处放行是明确打开的策略，得在结论里写清楚，
			// 否则日后追查「这条为什么没有出处还能自动定稿」会找不到原因。
			prefix = "自动定稿（无出处，按当前规则放行）："
		}
		return reviewResult{Verdict: reviewPass, Score: score, Note: prefix + buildReviewNote(score, good, nil)}
	}

	if refs == 0 && !rule.AutoAllowUnsourced {
		bad = append(bad, "无出处答案默认不放行")
	}
	return reviewResult{Verdict: reviewHold, Score: score, Note: "需人工：" + buildReviewNote(score, good, bad)}
}

// buildReviewNote 把得分依据拼成一句能看懂的话，存进 reviewNote 供审核台展示
func buildReviewNote(score int, good, bad []string) string {
	parts := make([]string, 0, len(good)+len(bad)+1)
	parts = append(parts, "得分 "+strconv.Itoa(score))
	if len(good) > 0 {
		parts = append(parts, strings.Join(good, "、"))
	}
	if len(bad) > 0 {
		parts = append(parts, strings.Join(bad, "、"))
	}
	return truncate(strings.Join(parts, "；"), 240)
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// applyReview 把结论落库。只处理待审核的条目：
// 已定稿、已下架、已合并的记录不能被一次自动审核改写状态。
func applyReview(item *model.KbQa, result reviewResult) bool {
	if item.Status != model.QaCandidate {
		return false
	}
	now := time.Now()
	fields := map[string]interface{}{
		"reviewScore":   result.Score,
		"reviewVerdict": result.Verdict,
		"reviewNote":    truncate(result.Note, 250),
		"reviewedBy":    "auto",
		"reviewedAt":    now,
	}
	switch result.Verdict {
	case reviewPass:
		fields["status"] = model.QaPublished
		item.Status = model.QaPublished
	case reviewReject:
		fields["status"] = model.QaRejected
		item.Status = model.QaRejected
		// 驳回理由写进备注，人工在「已驳回」里能直接看到为什么
		if strings.TrimSpace(item.Remark) == "" {
			fields["remark"] = truncate(result.Note, 200)
		}
	}
	if err := dao.UpdateKbQaFields(item.Id, fields); err != nil {
		log.Printf("写入审核结论失败 #%d: %v", item.Id, err)
		return false
	}
	item.ReviewScore = result.Score
	item.ReviewVerdict = result.Verdict
	item.ReviewNote = truncate(result.Note, 250)
	item.ReviewedBy = "auto"
	item.ReviewedAt = &now
	if result.Verdict == reviewPass {
		// 定稿集合变了，查重缓存立刻失效
		invalidatePublishedQa()
	}
	return true
}

// duplicateReject 重复条目的驳回结论
func duplicateReject(score int, exists string) reviewResult {
	return reviewResult{
		Verdict: reviewReject, Score: score,
		Note: "自动驳回：已存在相同问题的定稿条目「" + truncate(exists, 40) + "」，如内容更好请恢复后合并",
	}
}

// previewBatch 对一批候选做与正式审核完全一致的判定，返回 id -> 结论（不写库）。
//
// 为什么要按分数从高到低处理：同一个问题可能已经被问出好几条候选，
// 谁先定稿决定的是「哪条答案被留下」。按创建时间先到先得会让证据最弱的那条赢，
// 所以先打分排序，让出处齐全、内容最扎实的那条先占位，其余按重复驳回。
func previewBatch(ctx reviewCtx, list []model.KbQa) map[int64]reviewResult {
	scores := make(map[int64]reviewResult, len(list))
	for _, item := range list {
		scores[item.Id] = ctx.review(item)
	}
	ordered := make([]model.KbQa, len(list))
	copy(ordered, list)
	sort.SliceStable(ordered, func(i, j int) bool {
		return scores[ordered[i].Id].Score > scores[ordered[j].Id].Score
	})

	results := make(map[int64]reviewResult, len(list))
	for _, item := range ordered {
		result := scores[item.Id]
		key := normalizeQaQuestion(item.Question)
		if result.Verdict != reviewReject && ctx.published != nil && key != "" {
			if exists, ok := ctx.published[key]; ok {
				result = duplicateReject(result.Score, exists)
			}
		}
		results[item.Id] = result
		if result.Verdict == reviewPass && ctx.published != nil && key != "" {
			ctx.published[key] = item.Question
		}
	}
	return results
}

// autoReviewAfterCollect 沉淀入库后立即审一次。
// 不查重（这一条刚建，不可能和已定稿重复；与候选的重复在前面已经归并过），
// 所以只花一次内存计算，不产生额外查询。
func autoReviewAfterCollect(item *model.KbQa, rule kbRule) reviewResult {
	result := reviewCtx{rule: rule}.review(*item)
	applyReview(item, result)
	return result
}

// ---------- 接口 ----------

// qaAutoReview 批量自动审核。
// 传 id 只审这一条；不传则按待审核列表从新到旧扫一批，上限 500 条。
func qaAutoReview(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id    int64 `json:"id"`
		Limit int   `json:"limit"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	rule := loadKbRule()
	if !rule.AutoReview {
		return nil, badRequest("自动审核已关闭，请先到「知识规则」打开")
	}

	// 单条：用于编辑答案后重新判定
	if req.Id > 0 {
		item, err := dao.FindKbQa(req.Id)
		if err != nil {
			return nil, notFound("问答对不存在")
		}
		if item.Status != model.QaCandidate {
			return nil, badRequest("只有待审核的问答可以自动审核")
		}
		ctx := newReviewCtx(rule, true)
		result := ctx.review(*item)
		applyReview(item, result)
		return map[string]interface{}{
			"scanned": 1, "passed": boolToInt(result.Verdict == reviewPass),
			"rejected": boolToInt(result.Verdict == reviewReject),
			"held":     boolToInt(result.Verdict == reviewHold),
			"item":     buildQaItem(*item),
		}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	list, _, err := dao.ListKbQa(dao.QaQuery{Status: model.QaCandidate, Sort: "newest", Page: 1, Size: limit})
	if err != nil {
		return nil, serverError("查询候选失败: " + err.Error())
	}

	ctx := newReviewCtx(rule, true)
	results := previewBatch(ctx, list)
	passed, rejected, held := 0, 0, 0
	details := make([]map[string]interface{}, 0, 12)
	for i := range list {
		item := &list[i]
		result, ok := results[item.Id]
		if !ok {
			result = ctx.review(*item)
		}
		applyReview(item, result)
		switch result.Verdict {
		case reviewPass:
			passed++
		case reviewReject:
			rejected++
		default:
			held++
		}
		if len(details) < 12 {
			details = append(details, map[string]interface{}{
				"id": item.Id, "question": item.Question,
				"verdict": result.Verdict, "score": result.Score, "note": result.Note,
			})
		}
	}
	return map[string]interface{}{
		"scanned": len(list), "passed": passed, "rejected": rejected, "held": held,
		"score": rule.AutoScore, "allowUnsourced": rule.AutoAllowUnsourced,
		"details": details,
	}, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
