package service

import (
	"strings"
	"testing"
)

// TestAiSmallTalk 寒暄识别只做整句匹配。
// 这里最要紧的是「带着真问题的句子不能被当成打招呼」——
// 一旦误判，学员问「你好，G74 怎么用」就会收到一句自我介绍，问题被直接吞掉。
func TestAiSmallTalk(t *testing.T) {
	hit := []string{"你是谁", "你是谁？", "你叫什么名字", "介绍一下你自己", "你好", "Hi", "在吗", "你能做什么", "谢谢"}
	for _, q := range hit {
		if _, ok := aiSmallTalk(q); !ok {
			t.Errorf("应当识别为寒暄: %q", q)
		}
	}

	miss := []string{
		"你好，G74 怎么用",       // 带真问题
		"G74 端面切槽循环怎么用",    // 业务问题
		"谢谢，那 G74 的 P 是多少", // 致谢后继续追问
		"走心机断屑不好应该调哪个参数",
		"你是谁教出来的师傅", // 有实质内容，不按自我介绍回
	}
	for _, q := range miss {
		if _, ok := aiSmallTalk(q); ok {
			t.Errorf("不应识别为寒暄: %q", q)
		}
	}
}

// TestAiNoHitAnswer 未命中要分清「库是空的」「模型没启用」「资料里没这条」，
// 三种原因对应的下一步动作完全不同，混成一句话用户只能反复换问法。
func TestAiNoHitAnswer(t *testing.T) {
	if _, reason := aiNoHitAnswer("G70", 0, 0, true); reason != "kb_empty" {
		t.Errorf("空库应当返回 kb_empty，实际 %s", reason)
	}
	if _, reason := aiNoHitAnswer("G70", 3, 42, false); reason != "model_off" {
		t.Errorf("模型未启用应当返回 model_off，实际 %s", reason)
	}
	text, reason := aiNoHitAnswer("G70 是什么", 3, 42, true)
	if reason != "no_match" {
		t.Errorf("应当返回 no_match，实际 %s", reason)
	}
	if !strings.Contains(text, "3 份") || !strings.Contains(text, "42 个分块") {
		t.Errorf("说明里应当带上检索规模，实际 %q", text)
	}
	if !strings.Contains(text, "G70 是什么") {
		t.Errorf("说明里应当回显原问题，实际 %q", text)
	}
}
