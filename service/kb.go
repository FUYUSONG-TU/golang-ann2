package service

import (
	"net/http"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// 分块策略：每块约 500 字，尾块过短时并入上一块，避免出现只有一句话的碎片
const (
	kbChunkSize = 500
	kbChunkMin  = 80
)

// 单份资料正文上限，避免一次写入几十兆文本拖垮数据库
const kbContentLimit = 500000

// 版权归属中文文案
var kbCopyrightText = map[string]string{
	model.CopyrightOwn:        "自有",
	model.CopyrightAuthorized: "已授权",
	model.CopyrightPublic:     "公开资料",
	model.CopyrightUnknown:    "待确认",
}

// kbDefaultCategories 常用分类，避免每次都要手打
var kbDefaultCategories = []string{"工艺手册", "编程指令", "设备维护", "安全规范", "故障案例"}

// kbListItem 列表视图，附带版权与解析状态的中文文案
type kbListItem struct {
	model.KbDocument
	CopyrightText string `json:"copyrightText"`
	ParseText     string `json:"parseText"`
}

// kbList 资料列表
func kbList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	list, total, err := dao.ListKbDocuments(dao.KbQuery{
		Keyword:     strings.TrimSpace(r.URL.Query().Get("keyword")),
		Category:    strings.TrimSpace(r.URL.Query().Get("category")),
		Copyright:   strings.TrimSpace(r.URL.Query().Get("copyright")),
		ParseStatus: strings.TrimSpace(r.URL.Query().Get("parseStatus")),
		Status:      strings.TrimSpace(r.URL.Query().Get("status")),
		Page:        page,
		Size:        size,
	})
	if err != nil {
		return nil, serverError("查询资料失败: " + err.Error())
	}

	items := make([]kbListItem, 0, len(list))
	for _, item := range list {
		items = append(items, kbListItem{
			KbDocument:    item,
			CopyrightText: kbCopyrightText[item.Copyright],
			ParseText:     kbParseText(item.ParseStatus),
		})
	}

	parseCounts, _ := dao.CountKbByParseStatus()
	statusCounts, _ := dao.CountKbByStatus()
	categories, _ := dao.ListKbCategories()
	totals, _ := dao.SumKbTotals()

	return map[string]interface{}{
		"list":         items,
		"total":        total,
		"page":         page,
		"size":         size,
		"parseCounts":  parseCounts,
		"statusCounts": statusCounts,
		"categories":   mergeCategories(categories),
		"totals":       totals,
	}, nil
}

// kbDetail 资料详情，含正文
func kbDetail(c *adminCtx, r *http.Request) (interface{}, error) {
	id := queryInt64(r, "id", 0)
	if id <= 0 {
		return nil, badRequest("缺少资料 id")
	}
	item, err := dao.FindKbDocument(id)
	if err != nil {
		return nil, notFound("资料不存在")
	}
	return kbListItem{
		KbDocument:    *item,
		CopyrightText: kbCopyrightText[item.Copyright],
		ParseText:     kbParseText(item.ParseStatus),
	}, nil
}

// kbSave 新建或编辑资料，保存后立即解析一次
func kbSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id        int64  `json:"id"`
		Title     string `json:"title"`
		Category  string `json:"category"`
		Source    string `json:"source"`
		SourceUrl string `json:"sourceUrl"`
		Copyright string `json:"copyright"`
		Tags      string `json:"tags"`
		FileName  string `json:"fileName"`
		FileKey   string `json:"fileKey"`
		FileSize  int64  `json:"fileSize"`
		MimeType  string `json:"mimeType"`
		Content   string `json:"content"`
		Status    string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Category = strings.TrimSpace(req.Category)
	req.Source = strings.TrimSpace(req.Source)
	req.SourceUrl = strings.TrimSpace(req.SourceUrl)
	req.Tags = strings.TrimSpace(req.Tags)
	req.FileName = strings.TrimSpace(req.FileName)
	req.FileKey = strings.TrimSpace(req.FileKey)
	req.MimeType = strings.TrimSpace(req.MimeType)

	if req.Title == "" {
		return nil, badRequest("资料标题不能为空")
	}
	if len([]rune(req.Title)) > 128 {
		return nil, badRequest("资料标题不能超过 128 个字")
	}
	if len([]rune(req.Category)) > 32 {
		return nil, badRequest("分类不能超过 32 个字")
	}
	if len([]rune(req.Source)) > 64 {
		return nil, badRequest("来源不能超过 64 个字")
	}
	if len(req.SourceUrl) > 255 {
		return nil, badRequest("来源链接过长")
	}
	if len([]rune(req.Tags)) > 200 {
		return nil, badRequest("标签不能超过 200 个字，多个标签用逗号分隔")
	}
	if len([]rune(req.Content)) > kbContentLimit {
		return nil, badRequest("正文超过 50 万字，请拆成多份资料")
	}
	if req.Copyright == "" {
		req.Copyright = model.CopyrightUnknown
	}
	if _, ok := kbCopyrightText[req.Copyright]; !ok {
		return nil, badRequest("版权取值不合法")
	}
	if req.Status == "" {
		req.Status = model.ContentDraft
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}
	if req.Category == "" {
		req.Category = "未分类"
	}

	now := time.Now()
	var id int64
	if req.Id > 0 {
		if _, err := dao.FindKbDocument(req.Id); err != nil {
			return nil, notFound("资料不存在")
		}
		fields := map[string]interface{}{
			"title":     req.Title,
			"category":  req.Category,
			"source":    req.Source,
			"sourceUrl": req.SourceUrl,
			"copyright": req.Copyright,
			"tags":      req.Tags,
			"fileName":  req.FileName,
			"fileKey":   req.FileKey,
			"fileSize":  req.FileSize,
			"mimeType":  req.MimeType,
			"content":   req.Content,
			"updatedAt": now,
		}
		if err := dao.UpdateKbDocumentFields(req.Id, fields); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		id = req.Id
	} else {
		item := &model.KbDocument{
			Title:       req.Title,
			Category:    req.Category,
			Source:      req.Source,
			SourceUrl:   req.SourceUrl,
			Copyright:   req.Copyright,
			Tags:        req.Tags,
			FileName:    req.FileName,
			FileKey:     req.FileKey,
			FileSize:    req.FileSize,
			MimeType:    req.MimeType,
			Content:     req.Content,
			ParseStatus: model.ParsePending,
			Status:      model.ContentDraft, // 新建一律草稿，解析成功后再发布
			Sort:        dao.NextSort("kb_document"),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := dao.CreateKbDocument(item); err != nil {
			return nil, serverError("创建失败: " + err.Error())
		}
		id = item.Id
	}

	// 保存即解析：正文变化后字数与分块数必须重算，否则检索用的是旧数据
	if err := runKbParse(id); err != nil {
		return nil, err
	}

	// 只有解析成功的资料才允许直接以发布状态保存
	if req.Status == model.ContentPublished {
		saved, err := dao.FindKbDocument(id)
		if err != nil {
			return nil, serverError("读取资料失败: " + err.Error())
		}
		if saved.ParseStatus != model.ParseSuccess {
			return nil, badRequest("资料解析未通过，无法发布：" + saved.ParseError)
		}
		if err := dao.UpdateKbDocumentFields(id, map[string]interface{}{
			"status": model.ContentPublished, "updatedAt": time.Now(),
		}); err != nil {
			return nil, serverError("发布失败: " + err.Error())
		}
	} else if req.Id > 0 {
		if err := dao.UpdateKbDocumentFields(id, map[string]interface{}{
			"status": req.Status, "updatedAt": time.Now(),
		}); err != nil {
			return nil, serverError("保存状态失败: " + err.Error())
		}
	}
	// 新建时状态固定为草稿，保持与内容模块一致的状态机

	return dao.FindKbDocument(id)
}

// kbParseHandler 手动触发重新解析
func kbParseHandler(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少资料 id")
	}
	if _, err := dao.FindKbDocument(req.Id); err != nil {
		return nil, notFound("资料不存在")
	}
	if err := runKbParse(req.Id); err != nil {
		return nil, err
	}
	item, err := dao.FindKbDocument(req.Id)
	if err != nil {
		return nil, serverError("读取资料失败: " + err.Error())
	}
	if item.ParseStatus != model.ParseSuccess {
		// 解析失败也是"执行成功"，把原因回给前端展示，而不是抛 500
		return map[string]interface{}{
			"ok": false, "parseStatus": item.ParseStatus, "parseError": item.ParseError,
		}, nil
	}
	return map[string]interface{}{
		"ok": true, "parseStatus": item.ParseStatus,
		"charCount": item.CharCount, "chunkCount": item.ChunkCount,
	}, nil
}

// runKbParse 解析正文并回写字数、分块数、解析状态
func runKbParse(id int64) error {
	item, err := dao.FindKbDocument(id)
	if err != nil {
		return notFound("资料不存在")
	}
	content := strings.TrimSpace(item.Content)
	fields := map[string]interface{}{"updatedAt": time.Now()}

	if content == "" {
		fields["parseStatus"] = model.ParseFailed
		fields["charCount"] = 0
		fields["chunkCount"] = 0
		if item.FileKey != "" {
			fields["parseError"] = "文件已上传，但文件解析服务尚未接入，请先粘贴正文"
		} else {
			fields["parseError"] = "没有可解析的内容：正文为空且未上传文件"
		}
		fields["parsedAt"] = time.Now()
		return dao.UpdateKbDocumentFields(id, fields)
	}

	chars := len([]rune(content))
	chunks := buildKbChunks(item.Id, item.Title, item.Category, content)
	if err := dao.ReplaceKbChunks(item.Id, chunks); err != nil {
		// 分块写不进库就不能算解析成功，否则检索永远是空的
		fields["parseStatus"] = model.ParseFailed
		fields["parseError"] = "分块写入失败: " + truncate(err.Error(), 180)
		fields["charCount"] = chars
		fields["chunkCount"] = 0
		fields["parsedAt"] = time.Now()
		return dao.UpdateKbDocumentFields(id, fields)
	}
	fields["parseStatus"] = model.ParseSuccess
	fields["parseError"] = ""
	fields["charCount"] = chars
	fields["chunkCount"] = len(chunks)
	fields["parsedAt"] = time.Now()
	return dao.UpdateKbDocumentFields(id, fields)
}

// kbStatus 变更资料发布状态
func kbStatus(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id     int64  `json:"id"`
		Status string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少资料 id")
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}
	item, err := dao.FindKbDocument(req.Id)
	if err != nil {
		return nil, notFound("资料不存在")
	}
	if req.Status == model.ContentPublished {
		if item.ParseStatus != model.ParseSuccess {
			msg := "资料还没解析成功，不能发布"
			if item.ParseError != "" {
				msg += "：" + item.ParseError
			}
			return nil, badRequest(msg)
		}
		if item.Copyright == model.CopyrightUnknown {
			return nil, badRequest("版权归属还是「待确认」，确认来源后再发布")
		}
	}
	if err := dao.UpdateKbDocumentFields(req.Id, map[string]interface{}{
		"status": req.Status, "updatedAt": time.Now(),
	}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id, "status": req.Status}, nil
}

// kbDelete 删除资料
func kbDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少资料 id")
	}
	item, err := dao.FindKbDocument(req.Id)
	if err != nil {
		return nil, notFound("资料不存在")
	}
	if item.Status == model.ContentPublished {
		return nil, badRequest("已发布的资料先下架再删除，避免检索结果突然空掉")
	}
	// 分块是资料的派生数据，资料删除后必须一起清掉，否则会留下检索不到的孤儿块
	if err := dao.DeleteKbChunksByDoc(req.Id); err != nil {
		return nil, serverError("清理分块失败: " + err.Error())
	}
	if err := dao.DeleteKbDocument(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// kbChunks 查看某份资料的分块结果，用于排查「资料在库里但检索不到」的问题
func kbChunks(c *adminCtx, r *http.Request) (interface{}, error) {
	docId := queryInt64(r, "docId", 0)
	if docId <= 0 {
		return nil, badRequest("缺少资料 id")
	}
	list, err := dao.ListKbChunks(docId)
	if err != nil {
		return nil, serverError("查询分块失败: " + err.Error())
	}
	return map[string]interface{}{"list": list, "total": len(list)}, nil
}

// kbParseText 解析状态中文文案
func kbParseText(status string) string {
	switch status {
	case model.ParsePending:
		return "待解析"
	case model.ParseRunning:
		return "解析中"
	case model.ParseSuccess:
		return "解析成功"
	case model.ParseFailed:
		return "解析失败"
	}
	return status
}

// mergeCategories 合并预置分类与库中已有分类，去重后返回
func mergeCategories(used []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(kbDefaultCategories)+len(used))
	for _, item := range append(append([]string{}, kbDefaultCategories...), used...) {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

// buildKbChunks 把正文切成检索用的分块。
// 先按行（自然段）聚合到 500 字以内，超长段落再按句号拆开，
// 这样一块里尽量保持语义完整，而不是从句子中间截断。
func buildKbChunks(docId int64, title, category, content string) []model.KbChunk {
	pieces := packKbParagraphs(content)
	now := time.Now()
	chunks := make([]model.KbChunk, 0, len(pieces))
	for index, piece := range pieces {
		chunks = append(chunks, model.KbChunk{
			DocId:     docId,
			DocTitle:  title,
			Category:  category,
			Seq:       index + 1,
			Content:   piece,
			CharCount: len([]rune(piece)),
			CreatedAt: now,
		})
	}
	return chunks
}

// packKbParagraphs 按段落聚合成不超过 kbChunkSize 的文本块
func packKbParagraphs(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	out := make([]string, 0)
	var buf strings.Builder
	bufLen := 0
	flush := func() {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			out = append(out, text)
		}
		buf.Reset()
		bufLen = 0
	}

	for _, paragraph := range strings.Split(content, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		for _, piece := range splitLongKbParagraph(paragraph) {
			length := len([]rune(piece))
			if bufLen > 0 && bufLen+length > kbChunkSize {
				flush()
			}
			buf.WriteString(piece)
			buf.WriteString("\n")
			bufLen += length
		}
	}
	flush()

	// 尾块太短（如只剩一句）就并进上一块，减少无意义的碎片
	if len(out) > 1 && len([]rune(out[len(out)-1])) < kbChunkMin {
		last := out[len(out)-1]
		out = out[:len(out)-1]
		out[len(out)-1] = out[len(out)-1] + "\n" + last
	}
	return out
}

// splitLongKbParagraph 把超长段落按句末标点切开；没有标点时按两倍窗口强制切分
func splitLongKbParagraph(paragraph string) []string {
	runes := []rune(paragraph)
	if len(runes) <= kbChunkSize {
		return []string{paragraph}
	}
	out := make([]string, 0, len(runes)/kbChunkSize+1)
	buf := make([]rune, 0, kbChunkSize+32)
	for _, r := range runes {
		buf = append(buf, r)
		if len(buf) >= kbChunkSize && isKbSentenceEnd(r) {
			out = append(out, string(buf))
			buf = buf[:0]
			continue
		}
		if len(buf) >= kbChunkSize*2 {
			out = append(out, string(buf))
			buf = buf[:0]
		}
	}
	if len(buf) > 0 {
		out = append(out, string(buf))
	}
	return out
}

// isKbSentenceEnd 判断是否句末标点
func isKbSentenceEnd(r rune) bool {
	switch r {
	case '。', '！', '？', '；', '…', '.', '!', '?', ';':
		return true
	}
	return false
}
