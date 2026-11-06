package service

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

const (
	// qaHitThreshold 问答库直接命中的最低相似度。
	// 定得比文档检索高很多：文档命中只是「给模型一段参考」，
	// 问答库命中是「把已有答案原样交给学员」，错了就是直接错。
	qaHitThreshold = 0.45
	// qaDupThreshold 新问题并入已有问答对的相似度下限。
	// 比命中阈值略高：宁可多建一条候选，也不要把两个不同问题的答案混在一起。
	qaDupThreshold = 0.5
	// qaMatchPoolLimit 参与匹配的问答对上限。关键词匹配是线性扫描，
	// 超过这个量级要换成向量检索，否则每次提问都会明显变慢。
	qaMatchPoolLimit = 8000
	// qaMinAnswerRunes 发布前答案的最短长度，挡住「好的」「可以」这类空答案
	qaMinAnswerRunes = 10
	// qaAliasLimit 单条问答对最多挂多少个相似问法
	qaAliasLimit = 20
)

// 问答状态中文文案
var qaStatusText = map[string]string{
	model.QaCandidate: "待审核",
	model.QaPublished: "已定稿",
	model.QaOffline:   "已下架",
	model.QaMerged:    "已合并",
	model.QaRejected:  "已驳回",
}

// 问答来源中文文案
var qaSourceText = map[string]string{
	model.QaSourceAi:       "学员提问 · 有出处",
	model.QaSourceFallback: "AI 兜底 · 无出处",
	model.QaSourceManual:   "手工录入",
}

// ---------- 匹配 ----------

// qaStoredTerms 取问答对缓存的词元；老数据没缓存时现场切一次
func qaStoredTerms(item model.KbQa) []string {
	if cached := strings.TrimSpace(item.Terms); cached != "" {
		return strings.Fields(cached)
	}
	return aiQueryTerms(item.Question + " " + strings.ReplaceAll(item.Aliases, "\n", " "))
}

// qaBuildTerms 把问句与别名一起切词，结果存进 terms 字段供后续匹配复用
func qaBuildTerms(question, aliases string) string {
	text := question + " " + strings.ReplaceAll(aliases, "\n", " ")
	return strings.Join(aiQueryTerms(text), " ")
}

// qaMatchScore 问句相似度。
// 用词元集合的 Jaccard 系数衡量整体重叠，再叠加「查询词被覆盖的比例」：
// Jaccard 保证两个问句是同一件事，覆盖率保证换了一种说法的短问题也能召回。
// 最后乘一个长度比系数，避免用一个词去命中一整段长问题。
func qaMatchScore(item model.KbQa, querySet map[string]bool) float64 {
	// 单个词太模糊（「G74」「刀具」），交给文档检索让模型结合上下文回答，
	// 不要从问答库里硬凑一条出来。
	if len(querySet) < 2 {
		return 0
	}
	stored := qaStoredTerms(item)
	if len(stored) == 0 {
		return 0
	}
	storedSet := make(map[string]bool, len(stored))
	for _, term := range stored {
		storedSet[term] = true
	}
	intersection := 0
	for term := range querySet {
		if storedSet[term] {
			intersection++
		}
	}
	if intersection == 0 {
		return 0
	}

	union := len(querySet) + len(storedSet) - intersection
	jaccard := float64(intersection) / float64(union)
	coverage := float64(intersection) / float64(len(querySet))

	// 长度比越接近 1 越可信；系数区间只有 0.88 - 1.0，
	// 惩罚要轻——关键词式提问（「G74 端面切槽」）本来就比完整问句短。
	ratio := 1.0
	if len(querySet) > len(stored) && len(querySet) > 0 {
		ratio = float64(len(stored)) / float64(len(querySet))
	} else if len(stored) > len(querySet) && len(stored) > 0 {
		ratio = float64(len(querySet)) / float64(len(stored))
	}
	lengthFactor := 0.88 + 0.12*ratio

	// 覆盖率权重高于 Jaccard：学员换了说法时，问题里的词大多能被命中，
	// 但整体词元数量对不齐，Jaccard 会偏低。
	score := (jaccard*0.45 + coverage*0.55) * lengthFactor
	if score > 1 {
		score = 1
	}
	return score
}

// qaHit 一条问答库命中
type qaHit struct {
	Id       int64   `json:"id"`
	Question string  `json:"question"`
	Answer   string  `json:"answer"`
	Category string  `json:"category"`
	Score    float64 `json:"score"`
	HitCount int     `json:"hitCount"`
	Refs     string  `json:"-"`
}

// matchQa 在已定稿问答对里找最相似的问题。
// 只匹配 published：待审核的答案还没人确认过，不能直接交给学员。
func matchQa(query string, limit int) []qaHit {
	terms := aiQueryTerms(query)
	if len(terms) == 0 {
		return []qaHit{}
	}
	querySet := make(map[string]bool, len(terms))
	for _, term := range terms {
		querySet[term] = true
	}

	pool, err := dao.ListKbQaForMatch(qaMatchPoolLimit)
	if err != nil {
		log.Printf("读取问答库失败: %v", err)
		return []qaHit{}
	}

	hits := make([]qaHit, 0, 8)
	for _, item := range pool {
		if item.Status != model.QaPublished {
			continue
		}
		score := qaMatchScore(item, querySet)
		if score < qaHitThreshold {
			continue
		}
		hits = append(hits, qaHit{
			Id: item.Id, Question: item.Question, Answer: item.Answer,
			Category: item.Category, Score: score, HitCount: item.HitCount,
			Refs: item.Refs,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			// 同分时热度高的优先，同一个问题沉淀久了更可信
			return hits[i].HitCount > hits[j].HitCount
		}
		return hits[i].Score > hits[j].Score
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// findSimilarQa 在「已定稿 + 待审核」里找重复问题，用于沉淀时归并
func findSimilarQa(terms []string) (int64, float64, model.KbQa) {
	querySet := make(map[string]bool, len(terms))
	for _, term := range terms {
		querySet[term] = true
	}
	pool, err := dao.ListKbQaForMatch(qaMatchPoolLimit)
	if err != nil {
		return 0, 0, model.KbQa{}
	}
	bestId, bestScore := int64(0), 0.0
	var best model.KbQa
	for _, item := range pool {
		score := qaMatchScore(item, querySet)
		if score > bestScore {
			bestScore, bestId, best = score, item.Id, item
		}
	}
	return bestId, bestScore, best
}

// ---------- 沉淀 ----------

// collectQaCandidate 把一次「依据资料作答」的提问沉淀成问答对。
func collectQaCandidate(question, answer string, hits []aiHit) (int64, collectDecision) {
	return collectQaCandidateWithSource(question, answer, hits, model.QaSourceAi)
}

// collectQaCandidateWithSource 沉淀问答对，并记录答案的来源。
// 来源决定审核台的展示与「一键定稿」是否放行：无出处的兜底答案必须人工确认。
//
// 沉淀闸门设在这里而不是各个调用点：这是所有自动入库的必经之路，
// 放上游意味着以后新增一条入库路径就可能绕开规则，而规则一旦能被绕开就等于不存在。
func collectQaCandidateWithSource(question, answer string, hits []aiHit, source string) (int64, collectDecision) {
	question = strings.TrimSpace(question)
	answer = strings.TrimSpace(answer)
	if question == "" || answer == "" {
		return 0, collectDecision{Reason: collectTooShort, Message: "问题或答案为空"}
	}
	terms := aiQueryTerms(question)
	if len(terms) == 0 {
		return 0, collectDecision{Reason: collectTooShort, Message: "问题里没有可用于检索的关键词"}
	}

	// 先去重再看闸门：重复提问只是给已有条目计数，不新增数据，
	// 不该被「每日新增配额」误挡。
	if id, score, existed := findSimilarQa(terms); id > 0 && score >= qaDupThreshold {
		_ = dao.IncrKbQaCounter(id, "askCount", 1)
		appendQaAlias(existed, question)
		return id, collectDecision{
			Allowed: true, Reason: collectOK,
			Message: "已并入已有问答对 #" + strconv.FormatInt(id, 10) + "，提问次数 +1",
		}
	}

	rule := loadKbRule()
	if decision := rule.gate(question, answer, source); !decision.Allowed {
		log.Printf("问答未入库（%s）：%s | %s", decision.Reason, decision.Message, truncate(question, 60))
		return 0, decision
	}

	category := ""
	if len(hits) > 0 {
		category = hits[0].Category
	}
	refs, _ := json.Marshal(hits)
	// 兜底作答没有出处，统一存空数组而不是 null，避免审核台和前端多做一次空值判断
	if len(hits) == 0 {
		refs = []byte("[]")
	}
	if source == "" {
		source = model.QaSourceAi
	}
	now := time.Now()
	item := &model.KbQa{
		Question:  truncate(question, 240),
		Answer:    answer,
		Terms:     qaBuildTerms(question, ""),
		Category:  category,
		Source:    source,
		Refs:      string(refs),
		AskCount:  1,
		Status:    model.QaCandidate,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateKbQa(item); err != nil {
		log.Printf("沉淀问答对失败: %v", err)
		return 0, collectDecision{Reason: "create_failed", Message: "写入失败：" + err.Error()}
	}

	// 入库后立刻按规则审一次：能在沉淀这一步定下来的条目，
	// 就不该再堆到「待审核」里等人点。判定依据与批量审核完全同一套函数。
	if rule.AutoReview && rule.AutoOnCollect {
		result := autoReviewAfterCollect(item, rule)
		switch result.Verdict {
		case reviewPass:
			return item.Id, collectDecision{
				Allowed: true, Reason: collectOK,
				Message: "已按规则自动定稿（" + strconv.Itoa(result.Score) + " 分）",
			}
		case reviewReject:
			return item.Id, collectDecision{
				Allowed: true, Reason: collectOK,
				Message: "已入库但被自动审核驳回，理由见审核台",
			}
		default:
			return item.Id, collectDecision{
				Allowed: true, Reason: collectOK,
				Message: "已入候选池，自动审核未放行（" + strconv.Itoa(result.Score) + " 分），待人工复核",
			}
		}
	}
	return item.Id, collectDecision{Allowed: true, Reason: collectOK, Message: "已入候选池待核实"}
}

// appendQaAlias 把一个新问法并进已有问答对的别名表。
// 别名是「同一个问题的不同说法」，积累起来能显著提高后续的召回率。
func appendQaAlias(item model.KbQa, question string) {
	question = truncate(strings.TrimSpace(question), 200)
	if question == "" || question == item.Question {
		return
	}
	aliases := splitQaAliases(item.Aliases)
	if len(aliases) >= qaAliasLimit {
		return
	}
	for _, alias := range aliases {
		if alias == question {
			return
		}
	}
	aliases = append(aliases, question)
	updated := strings.Join(aliases, "\n")
	if err := dao.UpdateKbQaFields(item.Id, map[string]interface{}{
		"aliases": updated,
		"terms":   qaBuildTerms(item.Question, updated),
	}); err != nil {
		log.Printf("合并相似问法失败: %v", err)
	}
}

// splitQaAliases 别名按行存储，读取时去掉空行
func splitQaAliases(raw string) []string {
	out := make([]string, 0, 8)
	for _, line := range strings.Split(raw, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// parseQaRefs 反序列化出处，格式与文档命中的 aiHit 一致，前端可以复用同一套渲染
func parseQaRefs(raw string) []aiHit {
	out := make([]aiHit, 0, 4)
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []aiHit{}
	}
	return out
}

// qaSuggestPublish 判断候选是否够格直接发布：
// 必须带出处（挡住模型凭常识编的答案）、答案有实质内容、没被学员否定过，
// 并且有足够的真实需求（被反复问到）或正向反馈。
func qaSuggestPublish(item model.KbQa) bool {
	if item.Status != model.QaCandidate {
		return false
	}
	if len(parseQaRefs(item.Refs)) == 0 {
		return false
	}
	if len([]rune(strings.TrimSpace(item.Answer))) < qaMinAnswerRunes {
		return false
	}
	if item.DownCount > 0 {
		return false
	}
	return item.AskCount >= 3 || item.UpCount >= 2
}

// qaHeat 热度分，与 SQL 里的排序表达式保持一致
func qaHeat(item model.KbQa) int {
	return item.AskCount + item.HitCount*2 + item.UpCount*3 - item.DownCount*4
}

// ---------- 审核台 ----------

// qaListItem 审核台列表视图
type qaListItem struct {
	model.KbQa
	StatusText     string  `json:"statusText"`
	SourceText     string  `json:"sourceText"`
	Heat           int     `json:"heat"`
	SuggestPublish bool    `json:"suggestPublish"`
	RefList        []aiHit `json:"refList"`
	// 待审核条目带上一次自动审核的预判，页面不用点按钮就能看到它会怎么被判
	ReviewPreview *reviewResult `json:"reviewPreview,omitempty"`
}

func buildQaItem(item model.KbQa) qaListItem {
	return qaListItem{
		KbQa:           item,
		StatusText:     qaStatusText[item.Status],
		SourceText:     qaSourceText[item.Source],
		Heat:           qaHeat(item),
		SuggestPublish: qaSuggestPublish(item),
		RefList:        parseQaRefs(item.Refs),
	}
}

// qaList 问答对列表
func qaList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortBy == "" {
		sortBy = "heat"
	}
	list, total, err := dao.ListKbQa(dao.QaQuery{
		Keyword:  strings.TrimSpace(r.URL.Query().Get("keyword")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
		Source:   strings.TrimSpace(r.URL.Query().Get("source")),
		Sort:     sortBy,
		Page:     page,
		Size:     size,
	})
	if err != nil {
		return nil, serverError("查询问答库失败: " + err.Error())
	}

	rule := loadKbRule()
	// 预判用「同一批候选一起算」：批次内重复的条目会在这一步被判出来，
	// 所以页面上的预判和点「自动审核」后的真实结果一致，不会出现「显示可定稿、点了却被驳回」。
	reviewMap := map[int64]reviewResult{}
	suggestCount, autoPassCount, autoRejectCount := 0, 0, 0
	if candidates, _, err := dao.ListKbQa(dao.QaQuery{Status: model.QaCandidate, Page: 1, Size: 500}); err == nil {
		reviewMap = previewBatch(newReviewCtx(rule, true), candidates)
		for _, item := range candidates {
			if qaSuggestPublish(item) {
				suggestCount++
			}
			switch reviewMap[item.Id].Verdict {
			case reviewPass:
				autoPassCount++
			case reviewReject:
				autoRejectCount++
			}
		}
	}

	items := make([]qaListItem, 0, len(list))
	for _, item := range list {
		view := buildQaItem(item)
		if item.Status == model.QaCandidate {
			preview, ok := reviewMap[item.Id]
			if !ok {
				preview = previewReview(newReviewCtx(rule, true), item)
			}
			view.ReviewPreview = &preview
		}
		items = append(items, view)
	}
	statusCounts, _ := dao.CountKbQaByStatus()
	totals, _ := dao.SumKbQaTotals()
	categories, _ := dao.ListKbQaCategories()

	return map[string]interface{}{
		"list":            items,
		"total":           total,
		"page":            page,
		"size":            size,
		"statusCounts":    statusCounts,
		"totals":          totals,
		"categories":      categories,
		"suggestCount":    suggestCount,
		"autoReview":      rule.AutoReview,
		"autoScore":       rule.AutoScore,
		"autoPassCount":   autoPassCount,
		"autoRejectCount": autoRejectCount,
	}, nil
}

// qaSave 新建或编辑问答对。手工录入的条目直接来自运营，可信度高，
// 允许创建时就以定稿状态保存。
func qaSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id       int64  `json:"id"`
		Question string `json:"question"`
		Answer   string `json:"answer"`
		Aliases  string `json:"aliases"`
		Category string `json:"category"`
		Remark   string `json:"remark"`
		Status   string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	req.Question = strings.TrimSpace(req.Question)
	req.Answer = strings.TrimSpace(req.Answer)
	req.Category = strings.TrimSpace(req.Category)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Question == "" {
		return nil, badRequest("问题不能为空")
	}
	if len([]rune(req.Question)) > 200 {
		return nil, badRequest("问题不能超过 200 个字")
	}
	if req.Answer == "" {
		return nil, badRequest("答案不能为空")
	}
	if len([]rune(req.Category)) > 32 {
		return nil, badRequest("分类不能超过 32 个字")
	}
	if len([]rune(req.Remark)) > 200 {
		return nil, badRequest("备注不能超过 200 个字")
	}

	aliases := splitQaAliases(req.Aliases)
	if len(aliases) > qaAliasLimit {
		return nil, badRequest("相似问法最多 " + strconv.Itoa(qaAliasLimit) + " 条")
	}
	aliasText := strings.Join(aliases, "\n")
	terms := qaBuildTerms(req.Question, aliasText)
	if terms == "" {
		return nil, badRequest("问题里没有可用于检索的关键词")
	}

	if req.Id > 0 {
		if _, err := dao.FindKbQa(req.Id); err != nil {
			return nil, notFound("问答对不存在")
		}
		fields := map[string]interface{}{
			"question": req.Question,
			"answer":   req.Answer,
			"aliases":  aliasText,
			"terms":    terms,
			"category": req.Category,
			"remark":   req.Remark,
		}
		if err := dao.UpdateKbQaFields(req.Id, fields); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		item, err := dao.FindKbQa(req.Id)
		if err != nil {
			return nil, serverError("读取失败: " + err.Error())
		}
		return buildQaItem(*item), nil
	}

	status := req.Status
	if status == "" {
		status = model.QaCandidate
	}
	if _, ok := qaStatusText[status]; !ok {
		return nil, badRequest("状态取值不合法")
	}
	if status == model.QaMerged {
		return nil, badRequest("「已合并」是系统状态，不能手动指定")
	}
	now := time.Now()
	item := &model.KbQa{
		Question: req.Question, Answer: req.Answer, Aliases: aliasText, Terms: terms,
		Category: req.Category, Source: model.QaSourceManual, Status: status,
		Refs: "[]", Remark: req.Remark, CreatedAt: now, UpdatedAt: now,
	}
	if err := dao.CreateKbQa(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return buildQaItem(*item), nil
}

// qaStatus 变更问答对状态
func qaStatus(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id     int64  `json:"id"`
		Status string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少问答对 id")
	}
	item, err := dao.FindKbQa(req.Id)
	if err != nil {
		return nil, notFound("问答对不存在")
	}
	if _, ok := qaStatusText[req.Status]; !ok {
		return nil, badRequest("状态取值不合法")
	}
	if req.Status == model.QaMerged {
		return nil, badRequest("请使用合并操作，不要直接把状态改成「已合并」")
	}
	if item.Status == model.QaMerged {
		return nil, badRequest("该问答对已合并到其他条目，不能再改状态")
	}
	if req.Status == model.QaPublished {
		if len([]rune(strings.TrimSpace(item.Answer))) < qaMinAnswerRunes {
			return nil, badRequest("答案内容太短，定稿后学员会看到，请先补全")
		}
	}
	// 人工改状态时把审核痕迹一并写下来：
	// 「谁定的」比「定了什么」更重要——出事时要能追到是规则放行的还是人拍板的。
	now := time.Now()
	fields := map[string]interface{}{
		"status":     req.Status,
		"reviewedBy": "manual",
		"reviewedAt": now,
	}
	switch req.Status {
	case model.QaPublished:
		fields["reviewVerdict"] = reviewPass
		fields["reviewNote"] = "人工审核定稿"
	case model.QaRejected:
		fields["reviewVerdict"] = reviewReject
		fields["reviewNote"] = "人工审核驳回"
	}
	if err := dao.UpdateKbQaFields(req.Id, fields); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	// 定稿集合变了，自动审核的查重缓存要跟着失效
	if item.Status == model.QaPublished || req.Status == model.QaPublished || req.Status == model.QaOffline {
		invalidatePublishedQa()
	}
	return map[string]interface{}{"id": req.Id, "status": req.Status}, nil
}

// qaMerge 把重复问题并入目标问答对：
// 别名、热度、反馈全部累加过去，源记录标记为已合并并保留指向，便于追溯。
func qaMerge(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		FromId int64 `json:"fromId"`
		ToId   int64 `json:"toId"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.FromId <= 0 || req.ToId <= 0 {
		return nil, badRequest("缺少要合并的问答对")
	}
	if req.FromId == req.ToId {
		return nil, badRequest("不能和自己合并")
	}
	from, err := dao.FindKbQa(req.FromId)
	if err != nil {
		return nil, notFound("待合并的问答对不存在")
	}
	to, err := dao.FindKbQa(req.ToId)
	if err != nil {
		return nil, notFound("目标问答对不存在")
	}
	if from.Status == model.QaMerged {
		return nil, badRequest("该问答对已经合并过了")
	}
	if to.Status == model.QaMerged {
		return nil, badRequest("目标问答对已被合并，请换一条")
	}

	aliases := splitQaAliases(to.Aliases)
	for _, extra := range append([]string{from.Question}, splitQaAliases(from.Aliases)...) {
		if len(aliases) >= qaAliasLimit {
			break
		}
		duplicated := extra == to.Question
		for _, alias := range aliases {
			if alias == extra {
				duplicated = true
				break
			}
		}
		if !duplicated && strings.TrimSpace(extra) != "" {
			aliases = append(aliases, extra)
		}
	}
	aliasText := strings.Join(aliases, "\n")

	if err := dao.UpdateKbQaFields(req.ToId, map[string]interface{}{
		"aliases":   aliasText,
		"terms":     qaBuildTerms(to.Question, aliasText),
		"askCount":  to.AskCount + from.AskCount,
		"hitCount":  to.HitCount + from.HitCount,
		"upCount":   to.UpCount + from.UpCount,
		"downCount": to.DownCount + from.DownCount,
	}); err != nil {
		return nil, serverError("合并失败: " + err.Error())
	}
	if err := dao.UpdateKbQaFields(req.FromId, map[string]interface{}{
		"status": model.QaMerged, "mergedTo": req.ToId,
	}); err != nil {
		return nil, serverError("标记源问答对失败: " + err.Error())
	}
	return map[string]interface{}{"fromId": req.FromId, "toId": req.ToId}, nil
}

// qaBulkPublish 一键定稿所有「符合条件」的候选，把审核成本降到一次点击
func qaBulkPublish(c *adminCtx, r *http.Request) (interface{}, error) {
	list, _, err := dao.ListKbQa(dao.QaQuery{Status: model.QaCandidate, Page: 1, Size: 500})
	if err != nil {
		return nil, serverError("查询候选失败: " + err.Error())
	}
	published := 0
	for _, item := range list {
		if !qaSuggestPublish(item) {
			continue
		}
		if err := dao.UpdateKbQaFields(item.Id, map[string]interface{}{"status": model.QaPublished}); err != nil {
			continue
		}
		published++
	}
	if published > 0 {
		invalidatePublishedQa()
	}
	return map[string]interface{}{"published": published, "scanned": len(list)}, nil
}

// qaDelete 删除问答对
func qaDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少问答对 id")
	}
	if _, err := dao.FindKbQa(req.Id); err != nil {
		return nil, notFound("问答对不存在")
	}
	if err := dao.DeleteKbQa(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	invalidatePublishedQa()
	return map[string]interface{}{"id": req.Id}, nil
}

// qaFeedback 学员对答案的反馈。差评直接压低热度并在审核台标红，
// 让「越用越准」和「越用越歪」都能被及时发现。
func qaFeedback(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id      int64 `json:"id"`
		Helpful bool  `json:"helpful"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少问答对 id")
	}
	item, err := dao.FindKbQa(req.Id)
	if err != nil {
		return nil, notFound("问答对不存在")
	}
	field := "downCount"
	if req.Helpful {
		field = "upCount"
	}
	if err := dao.IncrKbQaCounter(req.Id, field, 1); err != nil {
		return nil, serverError("记录反馈失败: " + err.Error())
	}

	// 差评累计到 3 次且超过好评，自动下架：错答案留在库里比没有答案更糟
	down, up := item.DownCount+1, item.UpCount
	if !req.Helpful && down >= 3 && down > up && item.Status == model.QaPublished {
		_ = dao.UpdateKbQaFields(req.Id, map[string]interface{}{
			"status": model.QaOffline,
			"remark": "差评过多已自动下架，请复核答案",
		})
		return map[string]interface{}{"id": req.Id, "autoOffline": true}, nil
	}
	return map[string]interface{}{"id": req.Id, "autoOffline": false}, nil
}
