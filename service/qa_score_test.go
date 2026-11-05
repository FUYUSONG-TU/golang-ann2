package service

import (
	"testing"

	"wxcloudrun-golang/db/model"
)

// qaFixture 构造一条带词元缓存的问答对，模拟落库后的状态
func qaFixture(question, aliases string) model.KbQa {
	return model.KbQa{
		Question: question,
		Aliases:  aliases,
		Terms:    qaBuildTerms(question, aliases),
	}
}

func qaScoreOf(query string, item model.KbQa) float64 {
	terms := aiQueryTerms(query)
	set := make(map[string]bool, len(terms))
	for _, term := range terms {
		set[term] = true
	}
	return qaMatchScore(item, set)
}

func TestQaMatchScore(t *testing.T) {
	g74 := qaFixture("G74 端面切槽循环怎么用", "G74 怎么切端面槽")
	chip := qaFixture("走心机断屑不好应该调哪个参数", "")

	cases := []struct {
		name  string
		query string
		item  model.KbQa
		min   float64
		max   float64
	}{
		// 挂了别名之后 Jaccard 的分母会变大，原句也不可能拿到满分，
		// 0.8 以上说明「就是同一个问题」，与 0.45 的命中阈值留有充足余量。
		{"原句完全一致", "G74 端面切槽循环怎么用", g74, 0.8, 1.0},
		{"换一种说法", "G74端面切槽怎么用？", g74, 0.45, 1.0},
		{"关键词式提问", "G74 端面切槽", g74, 0.45, 1.0},
		{"别名命中", "G74 怎么切端面槽", g74, 0.45, 1.0},
		{"单词查询不硬凑", "G74", g74, 0, 0},
		{"无关问题不命中", "量子纠缠的退相干时间是多少", g74, 0, 0},
		{"另一个主题不误命中", "刀具磨损过快有哪些常见原因", g74, 0, 0.45},
		{"工艺问题换说法", "走心机断屑不好应该调什么参数", chip, 0.45, 1.0},
		{"空查询", "", g74, 0, 0},
	}

	for _, tc := range cases {
		got := qaScoreOf(tc.query, tc.item)
		if got < tc.min || got > tc.max {
			t.Errorf("%s：查询 %q 得分 %.3f，期望落在 [%.2f, %.2f]", tc.name, tc.query, got, tc.min, tc.max)
		}
	}
}

// TestQaMatchScoreRanking 同一问题换了说法，得分必须高于无关问题，
// 否则检索排序会把不相干的历史答案顶到第一位。
func TestQaMatchScoreRanking(t *testing.T) {
	g74 := qaFixture("G74 端面切槽循环怎么用", "")
	same := qaScoreOf("G74 端面切槽怎么用", g74)
	other := qaScoreOf("G74 钻孔循环怎么用", g74)
	if same <= other {
		t.Fatalf("同义问法 %.3f 应高于不同问法 %.3f", same, other)
	}
}

func TestQaBuildTerms(t *testing.T) {
	terms := qaBuildTerms("G74 端面切槽循环怎么用", "G74 怎么切端面槽")
	if terms == "" {
		t.Fatal("词元缓存不应为空，否则匹配永远返回 0")
	}
	// 缓存写入数据库的字段只有 512 字节，超了会被截断导致匹配失效
	if len(terms) > 500 {
		t.Fatalf("词元缓存过长（%d 字节），会被 terms 字段截断", len(terms))
	}
}
