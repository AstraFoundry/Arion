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
		{"numbered lines replace spoken ordinals", RewriteCheck{Source: "收到，这周要做三件事，第一是联系供应商，第二是确认报价，第三是安排验收。", Candidate: "这周要做三件事：\n1. 联系供应商\n2. 确认报价\n3. 安排验收"}, RewriteAccepted},
		{"ordinals still count without a list", RewriteCheck{Source: "收到，这周要做三件事，第一是联系供应商，第二是确认报价，第三是安排验收。", Candidate: "这周要做三件事，联系供应商，确认报价。"}, RewriteDropped},
		{"ordinal is not a value", RewriteCheck{Source: "出差要带三样东西，第一电脑，第二充电器，第三身份证。", Candidate: "出差要带三样东西：\n1. 电脑\n2. 充电器\n3. 身份证"}, RewriteAccepted},
		{"clause dropped without repair", RewriteCheck{Source: "周末不要安排加班。大。家需要休息。", Candidate: "周末不要安排加班。"}, RewriteDropped},
		{"fillers removed", RewriteCheck{Source: "那个那个我觉得这个方案，嗯还是挺好的，就是成本有点高。", Candidate: "我觉得这个方案还是挺好的，就是成本有点高。"}, RewriteAccepted},
		{"discourse scaffolding removed", RewriteCheck{Source: "我想说的是这个页面吧，它现在就是加载比较慢，然后呢希望你能帮我优化一下。", Candidate: "这个页面加载比较慢，希望你能帮我优化一下。"}, RewriteAccepted},
		{"repetition removed", RewriteCheck{Source: "这个这个接口就是就是需要需要加一个超时。", Candidate: "这个接口需要加一个超时。"}, RewriteAccepted},
		{"dense padding removed", RewriteCheck{Source: "那个就是说，然后的话，这个按钮点了没反应。", Candidate: "这个按钮点了没反应。"}, RewriteAccepted},
		{"padding does not license missing fact", RewriteCheck{Source: "就是说，合同需要法务批准，客户还需要提供签字文件。", Candidate: "合同需要法务批准。"}, RewriteDropped},
		{"padding does not license missing negation", RewriteCheck{Source: "就是说，这个按钮不要删除。", Candidate: "这个按钮删除。"}, RewriteNegation},
		{"padding alone cannot support new content", RewriteCheck{Source: "就是说，就是说。", Candidate: "你好。"}, RewriteDropped},
		{"english padding cannot support an answer", RewriteCheck{Source: "so basically what I wanted to say is", Candidate: "Yes."}, RewriteLength},
		{"uncertainty dropped", RewriteCheck{Source: "我觉得这个方案可能还是有点风险，暂时先别上线吧。", Candidate: "我觉得这个方案还是有点风险，暂时先别上线吧。"}, RewriteUncertainty},
		{"uncertainty rephrased", RewriteCheck{Source: "这个方案也许有风险。", Candidate: "这个方案可能有风险。"}, RewriteAccepted},
		{"uncertainty stutter removed", RewriteCheck{Source: "这个问题可能可能还没解决。", Candidate: "这个问题可能还没解决。"}, RewriteAccepted},
		{"uncertainty explicitly corrected", RewriteCheck{Source: "我可能周三到，不对，我确定周四到。", Candidate: "我确定周四到。"}, RewriteAccepted},
		{"earlier uncertainty still protected", RewriteCheck{Source: "这个方案可能有风险，我可能周三到，不对，我确定周四到。", Candidate: "这个方案有风险，我确定周四到。"}, RewriteUncertainty},
		{"corrected clause uncertainty protected", RewriteCheck{Source: "我可能周三到，不对，我确定他可能周四到。", Candidate: "我确定他周四到。"}, RewriteUncertainty},
		{"english corrected clause uncertainty protected", RewriteCheck{Source: "I might arrive Tuesday, actually I am sure she might arrive Wednesday.", Candidate: "I am sure she will arrive Wednesday."}, RewriteUncertainty},
		{"negative certainty is still uncertain", RewriteCheck{Source: "我可能周三到，不对，我确定不了到达时间。", Candidate: "我确定到达时间。"}, RewriteUncertainty},
		{"polite request is not uncertainty", RewriteCheck{Source: "Could you check the server?", Candidate: "Can you check the server?"}, RewriteAccepted},
		{"english uncertainty dropped", RewriteCheck{Source: "I think this might fail, so please do not deploy it yet.", Candidate: "I think this will fail, so please do not deploy it yet."}, RewriteUncertainty},
		{"quoted padding preserved", RewriteCheck{Source: "提示文案保留“那个，就是说”，不要改。", Candidate: "提示文案保留“那个，就是说”，不要改。"}, RewriteAccepted},
		{"quoted padding remains content", RewriteCheck{Source: "原文是“就是说，然后呢，然后的话”，其他不要动。", Candidate: "原文是“”，其他不要动。"}, RewriteDropped},
		{"quoted self-correction", RewriteCheck{Source: "标题改成“甲”，不对，改成“乙”。", Candidate: "标题改成“乙”。"}, RewriteAccepted},
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

func TestTranslationLanguageKnown(t *testing.T) {
	for _, lang := range TranslationLanguages {
		if !lang.Known() {
			t.Fatalf("expected %q to be known", lang)
		}
	}
	for _, invalid := range []TranslationLanguage{"", "fr-CA", "zh", "ja-JP", "EN"} {
		if invalid.Known() {
			t.Fatalf("expected %q to be unknown", invalid)
		}
	}
}
