package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// AI 配置全部存放在 sys_config 的 ai 分组下，键名白名单见 aiConfigKeys。
// 密钥在库里是明文，接口一律返回脱敏值；保存时如果前端回传的是脱敏串则不覆盖。
const aiGroup = "ai"

var aiConfigKeys = []string{
	"aiEnabled", "aiProvider", "aiBaseUrl", "aiApiKey", "aiChatModel", "aiEmbeddingModel",
	"aiTemperature", "aiMaxTokens", "aiTopK", "aiMinScore", "aiSystemPrompt", "aiRefuseText",
}

// 默认值：没有配置过也能跑起来，只是没有 Key 时不能真正回答
const (
	aiDefaultBaseUrl      = "https://api.deepseek.com"
	aiDefaultChatModel    = "deepseek-chat"
	aiDefaultSystemPrompt = "你是机匠学徒的机床加工助手，服务对象是走心机、数控车床与加工中心的编程与操作人员。" +
		"回答要具体、可执行，涉及参数时给出数值和单位，涉及代码时给出完整可用片段。" +
		"只能依据下面提供的资料回答；资料里没有的内容要直说没有，不要凭常识补全，也不要编造不存在的指令或参数。"
	aiDefaultRefuse = "知识库里暂时没有找到与这个问题相关的内容。可以换个说法再问，或者先把相关资料补充到知识库。"
	aiCallTimeout   = 60 * time.Second
	aiMaxContext    = 6 * kbChunkSize * 3 // 上下文上限，防止一次把整本手册塞给模型
)

// aiConfig AI 运行期配置
type aiConfig struct {
	Enabled        bool
	Provider       string
	BaseUrl        string
	ApiKey         string
	ChatModel      string
	EmbeddingModel string
	Temperature    float64
	MaxTokens      int
	TopK           int
	MinScore       float64
	SystemPrompt   string
	RefuseText     string
}

// aiTestState 最近一次连通性测试结果，仅存进程内存，重启后清空
type aiTestState struct {
	At      time.Time
	Ok      bool
	Message string
	Latency int64
}

var lastAiTest aiTestState

// loadAiConfig 读取 AI 配置，缺项用默认值兜底
func loadAiConfig() aiConfig {
	values, _ := dao.ConfigMap()
	pick := func(key, def string) string {
		if v, ok := values[key]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return def
	}
	cfg := aiConfig{
		Enabled:        values["aiEnabled"] == "true",
		Provider:       pick("aiProvider", "deepseek"),
		BaseUrl:        pick("aiBaseUrl", aiDefaultBaseUrl),
		ApiKey:         strings.TrimSpace(values["aiApiKey"]),
		ChatModel:      pick("aiChatModel", aiDefaultChatModel),
		EmbeddingModel: strings.TrimSpace(values["aiEmbeddingModel"]),
		Temperature:    clampFloat(parseFloat(values["aiTemperature"], 0.3), 0, 2),
		MaxTokens:      int(clampFloat(parseFloat(values["aiMaxTokens"], 1200), 64, 8192)),
		TopK:           int(clampFloat(parseFloat(values["aiTopK"], 5), 1, 20)),
		MinScore:       clampFloat(parseFloat(values["aiMinScore"], 0.15), 0, 1),
		SystemPrompt:   pick("aiSystemPrompt", aiDefaultSystemPrompt),
		RefuseText:     pick("aiRefuseText", aiDefaultRefuse),
	}
	return cfg
}

func parseFloat(raw string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
		return v
	}
	return def
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// maskApiKey 只保留首尾少量字符，避免密钥在页面上被截图带走
func maskApiKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 8 {
		return "****"
	}
	return string(runes[:3]) + "****" + string(runes[len(runes)-4:])
}

// aiConfigPayload 下发给前端的配置视图
type aiConfigPayload struct {
	Enabled        bool    `json:"enabled"`
	Provider       string  `json:"provider"`
	BaseUrl        string  `json:"baseUrl"`
	ApiKey         string  `json:"apiKey"`
	ChatModel      string  `json:"chatModel"`
	EmbeddingModel string  `json:"embeddingModel"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"maxTokens"`
	TopK           int     `json:"topK"`
	MinScore       float64 `json:"minScore"`
	SystemPrompt   string  `json:"systemPrompt"`
	RefuseText     string  `json:"refuseText"`
}

func (cfg aiConfig) payload() aiConfigPayload {
	return aiConfigPayload{
		Enabled:        cfg.Enabled,
		Provider:       cfg.Provider,
		BaseUrl:        cfg.BaseUrl,
		ApiKey:         maskApiKey(cfg.ApiKey),
		ChatModel:      cfg.ChatModel,
		EmbeddingModel: cfg.EmbeddingModel,
		Temperature:    cfg.Temperature,
		MaxTokens:      cfg.MaxTokens,
		TopK:           cfg.TopK,
		MinScore:       cfg.MinScore,
		SystemPrompt:   cfg.SystemPrompt,
		RefuseText:     cfg.RefuseText,
	}
}

// aiConfigView 配置页初始化数据：配置本体 + 知识库就绪情况 + 最近测试结果
func aiConfigView(c *adminCtx, r *http.Request) (interface{}, error) {
	cfg := loadAiConfig()
	statusCounts, _ := dao.CountKbByStatus()
	chunks, _ := dao.CountKbChunks()
	searchable, _ := dao.CountSearchableDocs()
	parseCounts, _ := dao.CountKbByParseStatus()

	test := map[string]interface{}{"at": nil, "ok": false, "message": "还没有测试过", "latencyMs": 0}
	if !lastAiTest.At.IsZero() {
		test = map[string]interface{}{
			"at": lastAiTest.At, "ok": lastAiTest.Ok,
			"message": lastAiTest.Message, "latencyMs": lastAiTest.Latency,
		}
	}

	return map[string]interface{}{
		"config":  cfg.payload(),
		"hasKey":  cfg.ApiKey != "",
		"ready":   cfg.Enabled && cfg.ApiKey != "" && searchable > 0,
		"default": map[string]interface{}{"baseUrl": aiDefaultBaseUrl, "chatModel": aiDefaultChatModel},
		"stats": map[string]interface{}{
			"docs":        statusCounts[model.ContentPublished] + statusCounts[model.ContentDraft] + statusCounts[model.ContentPending] + statusCounts[model.ContentOffline],
			"published":   statusCounts[model.ContentPublished],
			"chunks":      chunks,
			"searchable":  searchable,
			"parseFailed": parseCounts[model.ParseFailed],
		},
		"lastTest": test,
	}, nil
}

// aiConfigSave 保存 AI 配置。只接受白名单键，并对数值做范围校验
func aiConfigSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Enabled        *bool   `json:"enabled"`
		Provider       string  `json:"provider"`
		BaseUrl        string  `json:"baseUrl"`
		ApiKey         string  `json:"apiKey"`
		ChatModel      string  `json:"chatModel"`
		EmbeddingModel string  `json:"embeddingModel"`
		Temperature    float64 `json:"temperature"`
		MaxTokens      int     `json:"maxTokens"`
		TopK           int     `json:"topK"`
		MinScore       float64 `json:"minScore"`
		SystemPrompt   string  `json:"systemPrompt"`
		RefuseText     string  `json:"refuseText"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	baseUrl := strings.TrimRight(strings.TrimSpace(req.BaseUrl), "/")
	if baseUrl == "" {
		baseUrl = aiDefaultBaseUrl
	}
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		return nil, badRequest("接口地址必须以 http:// 或 https:// 开头")
	}
	chatModel := strings.TrimSpace(req.ChatModel)
	if chatModel == "" {
		chatModel = aiDefaultChatModel
	}
	if req.MaxTokens < 64 || req.MaxTokens > 8192 {
		return nil, badRequest("最大输出长度需在 64 - 8192 之间")
	}
	if req.TopK < 1 || req.TopK > 20 {
		return nil, badRequest("检索条数需在 1 - 20 之间")
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		return nil, badRequest("温度需在 0 - 2 之间")
	}
	if req.MinScore < 0 || req.MinScore > 1 {
		return nil, badRequest("最低相关度需在 0 - 1 之间")
	}
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if len([]rune(systemPrompt)) > 4000 {
		return nil, badRequest("系统提示词不能超过 4000 字")
	}
	if systemPrompt == "" {
		systemPrompt = aiDefaultSystemPrompt
	}

	enabled := "false"
	if req.Enabled != nil && *req.Enabled {
		enabled = "true"
	}

	items := map[string]string{
		"aiEnabled":        enabled,
		"aiProvider":       strings.TrimSpace(req.Provider),
		"aiBaseUrl":        baseUrl,
		"aiChatModel":      chatModel,
		"aiEmbeddingModel": strings.TrimSpace(req.EmbeddingModel),
		"aiTemperature":    strconv.FormatFloat(req.Temperature, 'f', 2, 64),
		"aiMaxTokens":      strconv.Itoa(req.MaxTokens),
		"aiTopK":           strconv.Itoa(req.TopK),
		"aiMinScore":       strconv.FormatFloat(req.MinScore, 'f', 2, 64),
		"aiSystemPrompt":   systemPrompt,
		"aiRefuseText":     strings.TrimSpace(req.RefuseText),
	}
	if items["aiProvider"] == "" {
		items["aiProvider"] = "deepseek"
	}
	if items["aiRefuseText"] == "" {
		items["aiRefuseText"] = aiDefaultRefuse
	}

	// 传回的是脱敏串说明用户没改密钥，跳过；传空串表示主动清除
	apiKey := strings.TrimSpace(req.ApiKey)
	if !strings.Contains(apiKey, "****") {
		items["aiApiKey"] = apiKey
	}

	for _, key := range aiConfigKeys {
		value, ok := items[key]
		if !ok {
			continue
		}
		if err := dao.SaveConfig(key, value); err != nil {
			return nil, serverError("保存 " + key + " 失败: " + err.Error())
		}
	}

	cfg := loadAiConfig()
	searchable, _ := dao.CountSearchableDocs()
	return map[string]interface{}{
		"config":     cfg.payload(),
		"hasKey":     cfg.ApiKey != "",
		"searchable": searchable,
	}, nil
}

// ---------- 模型调用 ----------

type aiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      aiMessage `json:"message"`
		FinishReason string    `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// aiEndpoint 拼接接口地址，兼容用户直接填到 /v1 或整条路径的情况
func aiEndpoint(baseUrl, path string) string {
	baseUrl = strings.TrimRight(strings.TrimSpace(baseUrl), "/")
	if strings.HasSuffix(baseUrl, path) {
		return baseUrl
	}
	return baseUrl + path
}

// aiChatCompletion 调用 OpenAI 兼容的对话补全接口
func aiChatCompletion(cfg aiConfig, messages []aiMessage, maxTokens int, temperature float64) (*aiChatResponse, int64, error) {
	body := map[string]interface{}{
		"model":       cfg.ChatModel,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
		"stream":      false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, 0, errors.New("构造请求失败")
	}

	endpoint := aiEndpoint(cfg.BaseUrl, "/chat/completions")
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, errors.New("接口地址不合法: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	client := &http.Client{Timeout: aiCallTimeout}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		// 网络层错误里可能带完整 URL，但不含密钥，可以原样返回
		return nil, latency, errors.New("请求模型服务失败: " + err.Error())
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result aiChatResponse
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, latency, fmt.Errorf("模型服务返回了无法解析的内容（HTTP %d）", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		msg := "HTTP " + strconv.Itoa(resp.StatusCode)
		if result.Error != nil && result.Error.Message != "" {
			msg += ": " + truncate(result.Error.Message, 200)
		}
		return nil, latency, errors.New(msg)
	}
	if result.Error != nil && result.Error.Message != "" {
		return nil, latency, errors.New(truncate(result.Error.Message, 200))
	}
	if len(result.Choices) == 0 {
		return nil, latency, errors.New("模型没有返回任何内容")
	}
	return &result, latency, nil
}

// aiTestHandler 连通性测试：发一句最短的请求，验证地址、密钥、模型三项是否可用
func aiTestHandler(c *adminCtx, r *http.Request) (interface{}, error) {
	cfg := loadAiConfig()
	if cfg.ApiKey == "" {
		lastAiTest = aiTestState{At: time.Now(), Ok: false, Message: "还没有填写 API Key"}
		return map[string]interface{}{
			"ok": false, "message": "还没有填写 API Key，保存后再测试", "latencyMs": 0,
		}, nil
	}

	messages := []aiMessage{{Role: "user", Content: "只回复 ok 两个字母"}}
	result, latency, err := aiChatCompletion(cfg, messages, 16, 0)
	if err != nil {
		lastAiTest = aiTestState{At: time.Now(), Ok: false, Message: err.Error(), Latency: latency}
		// 测试失败属于预期结果，不抛 500，把原因交给页面展示
		return map[string]interface{}{
			"ok": false, "message": err.Error(), "latencyMs": latency, "model": cfg.ChatModel,
		}, nil
	}

	reply := ""
	if len(result.Choices) > 0 {
		reply = strings.TrimSpace(result.Choices[0].Message.Content)
	}
	msg := "连接正常"
	lastAiTest = aiTestState{At: time.Now(), Ok: true, Message: msg, Latency: latency}
	return map[string]interface{}{
		"ok": true, "message": msg, "latencyMs": latency,
		"model": result.Model, "reply": truncate(reply, 60),
		"usage": result.Usage,
	}, nil
}

// ---------- 检索 ----------

// aiHit 一条检索命中
type aiHit struct {
	DocId    int64   `json:"docId"`
	DocTitle string  `json:"docTitle"`
	Category string  `json:"category"`
	Seq      int     `json:"seq"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
}

// 中文检索用二元切分，这些词单拎出来没有区分度，直接丢掉
var aiStopWords = map[string]bool{
	"怎么": true, "什么": true, "如何": true, "为什": true, "么办": true, "可以": true,
	"的是": true, "了吗": true, "这个": true, "那个": true, "请问": true, "一下": true,
	"我们": true, "你们": true, "他们": true, "没有": true, "是不": true, "不是": true,
	"以及": true, "还有": true, "或者": true, "但是": true, "因为": true, "所以": true,
	"the": true, "and": true, "for": true, "with": true, "how": true, "what": true,
}

// aiQueryTerms 把问题拆成检索词：中文切二元词组，英文数字按单词
func aiQueryTerms(query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	seen := map[string]bool{}
	terms := make([]string, 0, 24)

	add := func(term string) {
		if len(terms) >= 16 {
			return
		}
		if term == "" || seen[term] || aiStopWords[term] {
			return
		}
		if len([]rune(term)) < 2 {
			return
		}
		seen[term] = true
		terms = append(terms, term)
	}

	runes := []rune(query)
	cjk := make([]rune, 0, 8)
	latin := make([]rune, 0, 8)

	flushCjk := func() {
		if len(cjk) >= 2 {
			for i := 0; i+1 < len(cjk); i++ {
				add(string(cjk[i : i+2]))
			}
			// 2 - 4 字的整词也带上，命中时权重更高
			if len(cjk) <= 4 {
				add(string(cjk))
			}
		}
		cjk = cjk[:0]
	}
	flushLatin := func() {
		if len(latin) >= 2 {
			add(string(latin))
		}
		latin = latin[:0]
	}

	for _, r := range runes {
		switch {
		case r >= 0x4E00 && r <= 0x9FFF:
			flushLatin()
			cjk = append(cjk, r)
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			flushCjk()
			latin = append(latin, r)
		default:
			flushCjk()
			flushLatin()
		}
	}
	flushCjk()
	flushLatin()
	return terms
}

// scoreHit 给候选块打分：查询词覆盖率占七成，词频与标题命中占三成
func scoreHit(hit model.KbChunk, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	content := strings.ToLower(hit.Content)
	title := strings.ToLower(hit.DocTitle)
	matched := 0
	weight := 0.0
	for _, term := range terms {
		count := strings.Count(content, term)
		if count > 0 {
			matched++
			if count > 3 {
				count = 3
			}
			weight += 0.4 + float64(count)/3*0.6
		}
		if strings.Contains(title, term) {
			weight += 0.2
		}
	}
	if matched == 0 {
		return 0
	}
	// 拆出多个检索词时，只命中一个词基本是「时间」「加工」这类通用词造成的误召回，
	// 直接判 0，避免问无关问题时也硬凑出一段资料。
	// 阈值取 5，是为了不影响「G74 是什么意思」这种真关键词只有一个的短问题。
	if len(terms) >= 5 && matched < 2 {
		return 0
	}
	score := float64(matched)/float64(len(terms))*0.7 + weight/float64(len(terms))*0.3
	if score > 1 {
		score = 1
	}
	return score
}

// retrieveKb 检索知识库，返回命中块与候选数量
func retrieveKb(query string, topK int, minScore float64) ([]aiHit, int, error) {
	terms := aiQueryTerms(query)
	if len(terms) == 0 {
		return []aiHit{}, 0, nil
	}
	if topK <= 0 {
		topK = 5
	}
	candidates, err := dao.SearchKbChunks(terms, 800)
	if err != nil {
		return nil, 0, err
	}

	hits := make([]aiHit, 0, len(candidates))
	for _, item := range candidates {
		score := scoreHit(item, terms)
		if score < minScore {
			continue
		}
		hits = append(hits, aiHit{
			DocId: item.DocId, DocTitle: item.DocTitle, Category: item.Category,
			Seq: item.Seq, Content: item.Content, Score: score,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].DocId < hits[j].DocId
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, len(candidates), nil
}

// aiRetrieveHandler 只检索不调用模型，用来验证知识库召回是否正常（不消耗 token）
func aiRetrieveHandler(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Query    string  `json:"query"`
		TopK     int     `json:"topK"`
		MinScore float64 `json:"minScore"`
		Debug    bool    `json:"debug"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return nil, badRequest("请输入要检索的问题")
	}
	if len([]rune(req.Query)) > 300 {
		return nil, badRequest("问题太长，请控制在 300 字以内")
	}

	cfg := loadAiConfig()
	topK := req.TopK
	if topK <= 0 {
		topK = cfg.TopK
	}
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = cfg.MinScore
	}

	hits, candidates, err := retrieveKb(req.Query, topK, minScore)
	if err != nil {
		return nil, serverError("检索失败: " + err.Error())
	}
	out := map[string]interface{}{
		"hits": hits, "count": len(hits), "candidates": candidates,
		"topK": topK, "minScore": minScore,
	}
	if req.Debug {
		out["terms"] = aiQueryTerms(req.Query)
	}
	return out, nil
}

// aiChatHandler 知识库问答：先检索，再把命中内容作为唯一依据交给模型
func aiChatHandler(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Question string  `json:"question"`
		TopK     int     `json:"topK"`
		MinScore float64 `json:"minScore"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return nil, badRequest("请输入要问的问题")
	}
	if len([]rune(req.Question)) > 500 {
		return nil, badRequest("问题太长，请控制在 500 字以内")
	}

	cfg := loadAiConfig()
	if !cfg.Enabled {
		return nil, badRequest("AI 助手还没启用，请先在 AI 配置页打开开关")
	}
	if cfg.ApiKey == "" {
		return nil, badRequest("还没有配置 API Key，请先在 AI 配置页填写")
	}

	topK := req.TopK
	if topK <= 0 {
		topK = cfg.TopK
	}
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = cfg.MinScore
	}
	hits, candidates, err := retrieveKb(req.Question, topK, minScore)
	if err != nil {
		return nil, serverError("检索失败: " + err.Error())
	}

	// 没有命中就不调用模型，直接返回兜底话术，省一次 token
	if len(hits) == 0 {
		return map[string]interface{}{
			"answer": cfg.RefuseText, "model": cfg.ChatModel, "hitCount": 0,
			"candidates": candidates, "references": []aiHit{}, "usage": nil, "latencyMs": 0,
			"grounded": false,
		}, nil
	}

	context := buildAiContext(hits)
	messages := []aiMessage{
		{Role: "system", Content: cfg.SystemPrompt + "\n\n以下是知识库资料：\n" + context},
		{Role: "user", Content: req.Question},
	}
	result, latency, err := aiChatCompletion(cfg, messages, cfg.MaxTokens, cfg.Temperature)
	if err != nil {
		// 密钥无效、余额不足、模型名写错都属于配置问题，返回 4xx 让页面提示「去改配置」，
		// 而不是当成服务端故障
		if strings.Contains(err.Error(), "HTTP 401") || strings.Contains(err.Error(), "HTTP 403") {
			return nil, badRequest("模型服务鉴权失败，请检查 API Key：" + truncate(err.Error(), 180))
		}
		if strings.Contains(err.Error(), "HTTP 400") || strings.Contains(err.Error(), "HTTP 404") {
			return nil, badRequest("模型服务拒绝了请求，请检查接口地址与模型名：" + truncate(err.Error(), 180))
		}
		return nil, serverError("模型调用失败: " + err.Error())
	}

	answer := ""
	if len(result.Choices) > 0 {
		answer = strings.TrimSpace(result.Choices[0].Message.Content)
	}
	if answer == "" {
		answer = cfg.RefuseText
	}

	return map[string]interface{}{
		"answer":     answer,
		"model":      result.Model,
		"hitCount":   len(hits),
		"candidates": candidates,
		"references": hits,
		"usage":      result.Usage,
		"latencyMs":  latency,
		"grounded":   true,
	}, nil
}

// buildAiContext 把命中块拼成带编号的参考资料，编号与前端引用一致
func buildAiContext(hits []aiHit) string {
	var buf strings.Builder
	total := 0
	for index, hit := range hits {
		text := hit.Content
		if total+len(text) > aiMaxContext {
			text = truncate(text, aiMaxContext-total)
		}
		if text == "" {
			break
		}
		buf.WriteString("[")
		buf.WriteString(strconv.Itoa(index + 1))
		buf.WriteString("] 《")
		buf.WriteString(hit.DocTitle)
		buf.WriteString("》第 ")
		buf.WriteString(strconv.Itoa(hit.Seq))
		buf.WriteString(" 段\n")
		buf.WriteString(text)
		buf.WriteString("\n\n")
		total += len(text)
	}
	return buf.String()
}
