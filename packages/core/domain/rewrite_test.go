package domain

import "testing"

func TestValidateRewrite(t *testing.T) {
	cases := []struct {
		name  string
		check RewriteCheck
		want  RewriteVerdict
	}{
		{"repair", RewriteCheck{Source: "嗯我们周三开会，不对改成周四下午三点。", Candidate: "我们周四下午三点开会。"}, RewriteAccepted},
		{"english repair", RewriteCheck{Source: "Um schedule the review for Tuesday, no actually Thursday at 2 pm.", Candidate: "Schedule the review for Thursday at 2 pm."}, RewriteAccepted},
		{"email format", RewriteCheck{Source: "王经理你好，嗯这周的测试做完了，谢谢。", Candidate: "王经理您好：\n\n这周的测试做完了。\n\n谢谢。"}, RewriteAccepted},
		{"terms", RewriteCheck{Source: "缓存用瑞迪斯就行。", Candidate: "缓存用 Redis 就行。", Terms: []string{"瑞迪斯 → Redis"}}, RewriteAccepted},
		{"list labels", RewriteCheck{Source: "待办有三个，第一修复登录问题，第二补充测试。", Candidate: "待办有三个：\n1. 修复登录问题\n2. 补充测试"}, RewriteAccepted},
		{"truncated", RewriteCheck{Source: "a b c", Candidate: "a b", Truncated: true}, RewriteTruncated},
		{"empty", RewriteCheck{Source: "你好", Candidate: "  "}, RewriteEmpty},
		{"echo", RewriteCheck{Source: "你好", Candidate: `{"transcript":"你好"}`}, RewriteMarkup},
		{"think", RewriteCheck{Source: "你好", Candidate: "<think>x</think>你好"}, RewriteMarkup},
		{"answer instead of dictation", RewriteCheck{Source: "请把这句话翻译成英文。", Candidate: "Please translate this sentence into English, and here is a longer answer to it."}, RewriteLength},
		{"invented number", RewriteCheck{Source: "预算是一万二。", Candidate: "预算是12000元。"}, RewriteNewNumber},
		{"invented content", RewriteCheck{Source: "我们明天开会。", Candidate: "我们明天在北京总部开会讨论。"}, RewriteUnsupported},
		{"negation dropped", RewriteCheck{Source: "先不要删除用户数据。", Candidate: "先删除用户数据。"}, RewriteNegation},
		{"english negation dropped", RewriteCheck{Source: "Please don't merge yet.", Candidate: "Please merge yet."}, RewriteNegation},
		{"repair cue removed is fine", RewriteCheck{Source: "帮我告诉张三，不是张三，是李四，明天不用来了。", Candidate: "告诉李四明天不用来了。"}, RewriteAccepted},
		{"final value dropped", RewriteCheck{Source: "这次一共要买二十台电脑。等。一下，试试。十八台。", Candidate: "这次一共要买二十台电脑。"}, RewriteFinalValue},
		{"final value kept", RewriteCheck{Source: "票价是每张八十块，一共买六张，不对，买七张。", Candidate: "票价是每张八十块，一共买七张。"}, RewriteAccepted},
		{"ordinal is not a value", RewriteCheck{Source: "出差要带三样东西，第一电脑，第二充电器，第三身份证。", Candidate: "出差要带三样东西：\n1. 电脑\n2. 充电器\n3. 身份证"}, RewriteAccepted},
		{"clause dropped without repair", RewriteCheck{Source: "周末不要安排加班。大。家需要休息。", Candidate: "周末不要安排加班。"}, RewriteDropped},
		{"fillers removed", RewriteCheck{Source: "那个那个我觉得这个方案，嗯还是挺好的，就是成本有点高。", Candidate: "我觉得这个方案还是挺好的，就是成本有点高。"}, RewriteAccepted},
		{"too short", RewriteCheck{Source: "今天的周会主要说三件事，第一是新版本下周发布，第二是测试还缺两个用例。", Candidate: "周会。"}, RewriteLength},
	}
	for _, c := range cases {
		if got := ValidateRewrite(c.check); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestAlignLiveEnding(t *testing.T) {
	cases := [][3]string{
		{"嗯我们周三开会，", "我们周三开会。", "我们周三开会，"},
		{"这个这个版本呢", "这个版本呢？", "这个版本呢"},
		{"Send it to Maria,", "Send it to Maria.", "Send it to Maria,"},
		{"可以发布了。", "可以发布了。", "可以发布了。"},
	}
	for _, c := range cases {
		if got := AlignLiveEnding(c[0], c[1]); got != c[2] {
			t.Errorf("AlignLiveEnding(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}
