package domain

import "testing"

func TestDeliveryAllowsStructuredRewritingWithoutSemanticVeto(t *testing.T) {
	cases := []RewriteCheck{
		{Source: "小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。", Candidate: "- 小陈负责设计，周三交设计稿。\n- 小刘负责测试，周五完成。"},
		{Source: "周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。", Candidate: "周经理您好：\n\n我周四请假一天，已交接给小赵。麻烦批准，谢谢。"},
		{Source: "不要发", Candidate: "发"},
	}
	for _, c := range cases {
		if got := ValidateRewriteDelivery(c); got != RewriteAccepted {
			t.Fatalf("semantic veto retained: %s", got)
		}
	}
}
func TestDeliveryStillRejectsIncompleteTransport(t *testing.T) {
	for _, c := range []RewriteCheck{{Candidate: ""}, {Candidate: "正文", Truncated: true}, {Candidate: "<think>reasoning</think>"}, {Candidate: string([]byte{0xff})}} {
		if ValidateRewriteDelivery(c) == RewriteAccepted {
			t.Fatal("accepted incomplete/control output")
		}
	}
}
