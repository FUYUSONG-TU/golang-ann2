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
	"aiFallbackEnabled", "aiFallbackPrompt",
}

// 默认值：没有配置过也能跑起来，只是没有 Key 时不能真正回答
const (
	aiDefaultBaseUrl      = "https://api.deepseek.com"
	aiDefaultChatModel    = "deepseek-chat"
	aiDefaultSystemPrompt = "你是机匠学徒的机床加工助手，服务对象是走心机、数控车床与加工中心的编程与操作人员。" +
		"回答要具体、可执行，涉及参数时给出数值和单位，涉及代码时给出完整可用片段。" +
		"只能依据下面提供的资料回答；资料里没有的内容要直说没有，不要凭常识补全，也不要编造不存在的指令或参数。"
	aiDefaultRefuse = "知识库里暂时没有找到与这个问题相关的内容。可以换个说法再问，或者先把相关资料补充到知识库。"

	// aiDefaultFallbackPrompt 资料未命中时的兜底指令。
	// 关键区别：这份指令不再要求「只依据资料」，而是要求模型在无资料前提下
	// 老实交代不确定性——宁可给出判断路径，也不要编一个像模像样的切削参数。
	aiDefaultFallbackPrompt = "你是机匠学徒的机床加工助手，服务对象是走心机、数控车床与加工中心的编程与操作人员。" +
		"本次知识库没有检索到相关资料，你只能依靠通用知识作答。" +
		"要求：一、先一句话说明这是通用知识、未经本厂资料核实；" +
		"二、优先讲原理、判断方法和排查顺序，涉及具体切削用量、转速进给、刀具型号时，只给出常见取值范围并注明需按机床手册与实际工况确认；" +
		"三、不要编造不存在的 G 代码、M 代码或厂商专有参数；" +
		"四、如果这个问题本身就超出你的知识范围，直接说不确定，并说明需要查什么资料。"

	aiCallTimeout = 60 * time.Second
	aiMaxContext  = 6 * kbChunkSize * 3 // 上下文上限，防止一次把整本手册塞给模型
)

// aiFallbackBanner 兜底答案的固定前缀。
// 在服务端拼接而不是靠模型自觉，是因为「有没有依据」这件事直接关系到车间参数安全，
// 不能交给模型自己声明。
const aiFallbackBanner = "【未引用资料 · 模型通用知识】\n" +
	"资料库里没有收录这个问题，以下回答由模型根据通用知识生成，未经资料核实。\n" +
	"涉及切削用量、刀具型号、宏程序参数，请以机床手册与工艺卡为准。\n\n"

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
	// Fallback 资料未命中时是否允许模型凭通用知识作答并沉淀进候选池
	Fallback bool
	// FallbackPrompt 兜底作答时使用的系统指令
	FallbackPrompt string
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
		// 兜底作答默认开启：没有它，知识库空着的时候每个新问题都是死胡同。
		// 只有管理员显式关掉才会变成 false。
		Fallback:       values["aiFallbackEnabled"] != "false",
		FallbackPrompt: pick("aiFallbackPrompt", aiDefaultFallbackPrompt),
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
	Fallback       bool    `json:"fallbackEnabled"`
	FallbackPrompt string  `json:"fallbackPrompt"`
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
		Fallback:       cfg.Fallback,
		FallbackPrompt: cfg.FallbackPrompt,
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
		Fallback       *bool   `json:"fallbackEnabled"`
		FallbackPrompt string  `json:"fallbackPrompt"`
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
	fallbackPrompt := strings.TrimSpace(req.FallbackPrompt)
	if len([]rune(fallbackPrompt)) > 4000 {
		return nil, badRequest("兜底提示词不能超过 4000 字")
	}
	if fallbackPrompt == "" {
		fallbackPrompt = aiDefaultFallbackPrompt
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
		"aiFallbackPrompt": fallbackPrompt,
	}
	if items["aiProvider"] == "" {
		items["aiProvider"] = "deepseek"
	}
	if items["aiRefuseText"] == "" {
		items["aiRefuseText"] = aiDefaultRefuse
	}
	// 前端没传开关就保持库里的原值，避免「只改别的字段」时把兜底开关意外重置。
	// 兜底是默认开启的能力，只有显式传 false 才关。
	if req.Fallback != nil {
		if *req.Fallback {
			items["aiFallbackEnabled"] = "true"
		} else {
			items["aiFallbackEnabled"] = "false"
		}
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

// ---------- 不依赖资料的固定回答 ----------

// 寒暄与自我介绍用固定话术，不调模型。
// 理由有三：不花钱；不会被模型改写语气；这类问题不含业务事实，走检索只会得到
// 一句「资料里没找到」，学员第一句话就被劝退，会以为整个助手是坏的。
const (
	aiAnswerIdentity = "我是这套数控课程的知识助手，回答只依据已发布的课程资料——手册、工艺卡、操作规程这些。\n\n" +
		"你可以直接问编程指令、工艺参数或操作规范。资料里没写的内容我不会编，会直接告诉你没找到。"
	aiAnswerGreet = "你好，把问题直接发过来就行，比如「G74 端面切槽循环里 P 和 Q 分别代表什么」。\n\n" +
		"我只依据已发布的资料作答，资料里没有的会直说没有。"
	aiAnswerAbility = "我能做的事：\n\n" +
		"1. 依据资料回答问题，并标出引用的是哪一份、第几段；\n" +
		"2. 把学员问过的问题沉淀成标准问答，下次同样的问题直接命中，不用再算一遍；\n" +
		"3. 在「检索调试」里显示一次检索到底命中了哪些分块，方便排查答非所问。\n\n" +
		"资料里没有的内容我不会编。"
	aiAnswerThanks = "不客气，还有别的问题尽管问。"
)

// aiSmallTalk 识别寒暄与自我介绍。
// 只做整句匹配：像「你好，G74 怎么用」这种带着真问题的句子必须放过去走检索，
// 否则会把有意义的问题当成打招呼回掉。
func aiSmallTalk(question string) (string, bool) {
	text := strings.ToLower(strings.TrimSpace(question))
	text = strings.Trim(text, " 　\t\r\n，。,.!！?？~～、;；:：")
	switch text {
	case "你是谁", "你是谁啊", "你叫什么", "你叫什么名字", "你是什么", "你是ai吗", "你是机器人吗",
		"介绍一下你自己", "自我介绍", "你是做什么的":
		return aiAnswerIdentity, true
	case "你好", "您好", "hi", "hello", "在吗", "在不在", "有人吗":
		return aiAnswerGreet, true
	case "你能做什么", "你会什么", "你能干什么", "你有什么功能", "有什么功能", "你能帮我做什么", "怎么用你":
		return aiAnswerAbility, true
	case "谢谢", "谢谢你", "感谢", "辛苦了":
		return aiAnswerThanks, true
	}
	return "", false
}

// aiNoHitAnswer 没命中时要说清三件事：为什么没命中、检索了什么、下一步做什么。
// 一律回同一句兜底话术，等于把「库是空的」「这题资料里没写」「模型没启用」三种
// 完全不同的原因混成一句，用户唯一的反应就是不停换问法。
func aiNoHitAnswer(question string, docs, chunks int64, modelReady bool) (string, string) {
	if chunks == 0 {
		// 有资料但没切出分块：多半是解析失败或正文为空，和「一份都没传」是两回事
		if docs > 0 {
			return "已发布 " + strconv.FormatInt(docs, 10) + " 份资料，但还没有解析出可检索的正文（0 个分块）。\n\n" +
				"到「资料管理」检查这几份资料的解析状态，重新解析并发布后再来问。", "kb_empty"
		}
		return "知识库现在还是空的（0 份已发布资料），任何问题都没法基于资料回答。\n\n" +
			"先到「资料管理」把手册、工艺卡或操作规程上传并发布，再回来问。资料里没写的内容我会直说没有，不会编。", "kb_empty"
	}
	if !modelReady {
		return "这个问题在资料里没有找到对应内容（已收录 " + strconv.FormatInt(docs, 10) + " 份资料、" +
			strconv.FormatInt(chunks, 10) + " 个分块）。\n\n" +
			"AI 问答目前是关闭的，没法让模型帮着换个角度理解问题——到「AI 配置」打开开关并填写 Key 后再试。", "model_off"
	}
	text := "资料库里没有找到与「" + truncate(question, 40) + "」相关的内容。已检索 " +
		strconv.FormatInt(docs, 10) + " 份已发布资料、" + strconv.FormatInt(chunks, 10) + " 个分块。\n\n"
	text += "可以换成资料里的原词再问一次（比如把「切槽」说成「端面切槽循环」），或者请管理员把这部分内容补进资料库。"
	return text, "no_match"
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
	// 顺带把问答库的命中情况一起返回，调试时能看出「为什么这次又调了模型」
	qaHits := matchQa(req.Query, 3)
	// 带上知识库规模：命中 0 条时能立刻分辨是「资料里没这条」还是「库根本是空的」
	docs, _ := dao.CountSearchableDocs()
	chunks, _ := dao.CountKbChunks()
	out := map[string]interface{}{
		"hits": hits, "count": len(hits), "candidates": candidates,
		"topK": topK, "minScore": minScore,
		"qaHits": qaHits, "qaThreshold": qaHitThreshold,
		"kbDocs": docs, "kbChunks": chunks,
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

	// 寒暄和自我介绍不依赖资料，直接给固定话术：0 token、语气稳定，
	// 也不会因为知识库还没录资料就先泼一盆「没找到」。
	if answer, ok := aiSmallTalk(req.Question); ok {
		return map[string]interface{}{
			"answer": answer, "model": "", "source": "guide", "reason": "smalltalk",
			"hitCount": 0, "candidates": 0, "references": []aiHit{},
			"usage": nil, "latencyMs": 0, "grounded": false,
		}, nil
	}

	// 第一步先查问答库：已定稿的答案直接原样返回，不消耗 token，也不会被模型改写。
	// 这条路径不依赖模型，所以放在开关校验之前——即使 AI 关闭或没配 Key，
	// 沉淀好的答案也应该继续服务学员，这是知识库和模型解耦的意义。
	// 同一个问题问第二遍，成本从一次模型调用降到一次查询。
	if qaHits := matchQa(req.Question, 3); len(qaHits) > 0 {
		top := qaHits[0]
		_ = dao.IncrKbQaCounter(top.Id, "hitCount", 1)
		refs := parseQaRefs(top.Refs)
		return map[string]interface{}{
			"answer":     top.Answer,
			"model":      "问答沉淀",
			"source":     "qa",
			"qaId":       top.Id,
			"qaQuestion": top.Question,
			"qaScore":    top.Score,
			"qaHitTotal": top.HitCount + 1,
			"hitCount":   len(refs),
			"candidates": len(qaHits),
			"references": refs,
			"usage":      nil,
			"latencyMs":  0,
			"grounded":   true,
		}, nil
	}

	cfg := loadAiConfig()
	modelReady := cfg.Enabled && cfg.ApiKey != ""

	topK := req.TopK
	if topK <= 0 {
		topK = cfg.TopK
	}
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = cfg.MinScore
	}

	// 检索本身不需要模型，所以先检索再判断是否可用：
	// 知识库空着的时候，报「资料里没找到」是错的，真正原因是压根没有资料。
	hits, candidates, err := retrieveKb(req.Question, topK, minScore)
	if err != nil {
		return nil, serverError("检索失败: " + err.Error())
	}

	// 没有命中时不再直接回「没找到」：模型可用且兜底开关打开，就让模型凭通用知识作答，
	// 答案带未核实声明后进候选池。这是知识库从 0 到 1 的入口——
	// 库空着也能靠真实提问攒出第一批问答对，审核定稿后就变成可直达的知识。
	if len(hits) == 0 {
		docs, _ := dao.CountSearchableDocs()
		chunks, _ := dao.CountKbChunks()

		// 领域准入放在这里：检索之后、调模型之前。
		// 检索命中过的内容天然是领域内资料，不需要再判；没命中的问题必须先过闸，
		// 否则「今天天气怎么样」也会被模型认真答一遍并写进候选池。
		rule := loadKbRule()
		verdict := rule.judge(req.Question)
		if !verdict.In && rule.Policy == kbPolicyStrict {
			return aiOutOfDomainAnswer(rule, verdict, docs, chunks), nil
		}

		if modelReady && cfg.Fallback {
			return aiFallbackAnswer(cfg, rule, req.Question, candidates, docs, chunks)
		}
		// 兜底关闭或模型不可用，才能退回说明原因的话术
		answer, reason := aiNoHitAnswer(req.Question, docs, chunks, modelReady)
		return map[string]interface{}{
			"answer": answer, "model": "", "source": "none", "reason": reason,
			"hitCount": 0, "candidates": candidates, "references": []aiHit{},
			"usage": nil, "latencyMs": 0, "grounded": false, "domain": verdict,
			"kbDocs": docs, "kbChunks": chunks,
		}, nil
	}

	// 命中但没有可用的模型：这时候确实有资料，只是组织不成回答，单独提示
	if !modelReady {
		return nil, badRequest("这个问题在资料里找到了依据，但 AI 问答还没启用（或没配 Key），先到 AI 配置页打开再问")
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

	// 答完立刻沉淀成候选问答对。有出处的答案才有资格进池子，
	// 进池后还要人工确认一次才会参与「直接命中」，避免错答案被反复引用。
	// 沉淀失败不影响本次回答，只记日志。
	qaId, decision := collectQaCandidate(req.Question, answer, hits)

	return map[string]interface{}{
		"answer":     answer,
		"model":      result.Model,
		"source":     "model",
		"qaId":       qaId,
		"hitCount":   len(hits),
		"candidates": candidates,
		"references": hits,
		"usage":      result.Usage,
		"latencyMs":  latency,
		"grounded":   true,
		"collect":    decision,
	}, nil
}

// aiFallbackAnswer 资料未命中时的兜底作答。
// 与主链路只有两点不同：不给模型资料上下文，答案前强制加「未核实」声明。
// 沉淀时来源记 ai_fallback 且 refs 为空——审核台的「一键定稿」只认有出处的条目，
// 所以这类答案必须人工逐条确认才会进入直接命中，不会被模型的自说自话污染。
// 此外还要过一道沉淀闸门：领域外、太短、像拒答、超出当日配额的答案一律不入库。
func aiFallbackAnswer(cfg aiConfig, rule kbRule, question string, candidates int, docs, chunks int64) (interface{}, error) {
	// 同一个问题可能已经被兜底答过一次，候选池里直接复用，省一次模型调用。
	if id, score, existed := findSimilarQa(aiQueryTerms(question)); id > 0 && score >= qaDupThreshold &&
		existed.Status == model.QaCandidate && len(parseQaRefs(existed.Refs)) == 0 {
		_ = dao.IncrKbQaCounter(id, "askCount", 1)
		return map[string]interface{}{
			"answer": aiFallbackBanner + existed.Answer, "model": "",
			"source": model.QaSourceFallback, "reason": "fallback_cached",
			"qaId": id, "qaQuestion": existed.Question,
			"hitCount": 0, "candidates": candidates, "references": []aiHit{},
			"usage": nil, "latencyMs": 0, "grounded": false, "verified": false,
			"kbDocs": docs, "kbChunks": chunks,
		}, nil
	}

	// 把领域范围写进指令：即便准入判定放行了，也要让模型知道自己的职责边界，
	// 避免它借着「通用知识」顺着问题往外滑。
	systemPrompt := cfg.FallbackPrompt
	if scope := strings.TrimSpace(rule.Scope); scope != "" {
		systemPrompt += "\n\n本知识库的收录范围（超出范围的内容直接说明不负责，不要展开）：\n" + scope
	}

	messages := []aiMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}
	result, latency, err := aiChatCompletion(cfg, messages, cfg.MaxTokens, cfg.Temperature)
	if err != nil {
		// 兜底失败不能再抛 500：学员要的是「这题现在答不了，以及为什么」，
		// 而不是一个红色报错弹窗。
		return map[string]interface{}{
			"answer": "这个问题暂时答不上来：资料库里没有收录，模型兜底作答也没有成功。\n\n" +
				"原因：" + truncate(err.Error(), 160) + "\n\n" +
				"可以到「AI 配置」检查接口地址与密钥，或者把相关资料补进知识库后重试。",
			"model": "", "source": "none", "reason": "fallback_failed",
			"hitCount": 0, "candidates": candidates, "references": []aiHit{},
			"usage": nil, "latencyMs": latency, "grounded": false,
			"kbDocs": docs, "kbChunks": chunks,
		}, nil
	}

	answer := ""
	if len(result.Choices) > 0 {
		answer = strings.TrimSpace(result.Choices[0].Message.Content)
	}
	if answer == "" {
		answer = cfg.RefuseText
	}
	// 入库只存答案本体：未核实声明是这次回答的属性，不是知识本身。
	// 人工核实定稿后，学员看到的就是干净的答案。
	// 入不入库由闸门决定，闸门理由原样返回，方便在调试台看清「这次为什么没沉淀」。
	qaId, decision := collectQaCandidateWithSource(question, answer, nil, model.QaSourceFallback)

	return map[string]interface{}{
		"answer": aiFallbackBanner + answer, "model": result.Model,
		"source": model.QaSourceFallback, "reason": "fallback", "qaId": qaId,
		"hitCount": 0, "candidates": candidates, "references": []aiHit{},
		"usage": result.Usage, "latencyMs": latency, "grounded": false, "verified": false,
		"collect": decision, "kbDocs": docs, "kbChunks": chunks,
	}, nil
}

// aiOutOfDomainAnswer 领域外问题的固定回复。
// 关键点是「什么都不做」：不调模型（省钱，也不给它跑偏的机会）、不写库（不污染知识库），
// 只把范围和原因说清楚。含糊其辞地答一半，比明确拒绝更糟。
func aiOutOfDomainAnswer(rule kbRule, verdict domainVerdict, docs, chunks int64) map[string]interface{} {
	var buf strings.Builder
	buf.WriteString("这个问题不在本知识库的收录范围内，我没有作答，也没有把它写进知识库。\n\n")
	buf.WriteString("我只负责「" + rule.Name + "」相关的问题——")
	buf.WriteString("设备结构与调整、刀具与磨损、切削参数、编程指令、装夹对刀、断屑排屑、质量检测、保养与故障处理。\n\n")
	if verdict.Reason == domainInjection {
		buf.WriteString("另外，试图改写助手指令的请求不会被处理。\n\n")
	} else if len(verdict.Blocked) > 0 {
		buf.WriteString("本次提问命中了范围外词汇：" + strings.Join(verdict.Blocked, "、") + "。\n\n")
	}
	buf.WriteString("如果是机加工相关的问题，换成具体的工序或现象再问一次，比如「不锈钢外圆车削振刀怎么调」。")

	return map[string]interface{}{
		"answer": buf.String(), "model": "", "source": "out_of_domain", "reason": verdict.Reason,
		"domain": verdict, "hitCount": 0, "candidates": 0, "references": []aiHit{},
		"usage": nil, "latencyMs": 0, "grounded": false, "verified": false,
		"kbDocs": docs, "kbChunks": chunks,
	}
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
