# Builds the held-out scene corpus. Written before live-v3 demonstrations;
# its hash is recorded so later prompt edits cannot silently tune on it.
import json, hashlib, sys

def case(id, lang, scene, target, source, ref, req=(), forb=(), prot=(), tags=()):
    return {"id": id, "language": lang, "speech": "", "sourceText": source, "references": [ref],
            "requiredStrings": list(req), "forbiddenStrings": list(forb), "protected": list(prot),
            "scenario": scene, "target": target, "context": "", "approvedMemory": [], "mode": "clean",
            "tags": ["scene-" + scene, *tags]}

T = lambda app="", field="text", site="": {"app": app, "field": field, "site": site}
C = "com.google.Chrome"; S = "com.apple.Safari"; E = "com.microsoft.edgemac"
samples = [
  # email
  case("se-zh-defense", "zh", "email", T("com.apple.mail"), "李老师您好，嗯，关于下周的答辩，我想把时间从周三改到周四下午，不知道您是否方便，谢谢。",
       "李老师您好：\n\n关于下周的答辩，我想把时间从周三改到周四下午，不知道您是否方便。\n\n谢谢。", ["\n\n", "周四下午"], ["嗯"], ["李老师"]),
  case("se-en-contract", "en", "email", T(C, site="mail.google.com"), "Hi Maria, um, thanks for sending the contract. I've reviewed it and uh everything looks fine except section four. Best, Tom.",
       "Hi Maria,\n\nThanks for sending the contract. I've reviewed it and everything looks fine except section four.\n\nBest,\nTom", ["\n\n", "section four"], ["um,", " uh "], ["Maria", "Tom"]),
  case("se-mx-roadmap", "mixed", "email", T("com.microsoft.Outlook"), "Hi Kevin，嗯，附件是 Q3 的 roadmap，麻烦周五之前给我 feedback，谢谢。",
       "Hi Kevin，\n\n附件是 Q3 的 roadmap，麻烦周五之前给我 feedback。\n\n谢谢。", ["\n\n", "Q3", "roadmap", "feedback"], ["嗯"], ["Kevin"]),
  case("se-zh-quote", "zh", "email", T(S, site="mail.qq.com"), "王总好，那个这次的报价单我已经更新了，总价是三万八千元，请您查收。",
       "王总好：\n\n这次的报价单我已经更新了，总价是三万八千元，请您查收。", ["\n\n"], ["那个这次"], ["王总"]),
  case("se-en-closure", "en", "email", T(E, site="outlook.office.com"), "Dear team, I wanted to let you know that the office will be closed on Monday. Please uh plan accordingly. Regards, Anna.",
       "Dear team,\n\nI wanted to let you know that the office will be closed on Monday. Please plan accordingly.\n\nRegards,\nAnna", ["\n\n", "Monday"], [" uh "], ["Anna"]),
  case("se-zh-invoice", "zh", "email", T("com.readdle.SparkDesktop"), "张经理，您好，呃，上次提到的那个发票，财务说还没收到，麻烦再发一次，谢谢。",
       "张经理，您好：\n\n上次提到的那个发票，财务说还没收到，麻烦再发一次。\n\n谢谢。", ["\n\n", "发票"], ["呃"], ["张经理"]),
  case("se-zh-repair", "zh", "email", T("com.apple.mail"), "陈老师您好，会议定在周二，不对，周三上午十点，请您准时参加，谢谢。",
       "陈老师您好：\n\n会议定在周三上午十点，请您准时参加。\n\n谢谢。", ["\n\n", "周三上午十点"], ["周二", "不对"], ["陈老师"], ["self-correction"]),
  # work
  case("sw-zh-three", "zh", "work", T("com.electron.lark"), "收到，嗯，今天要处理三件事，第一是修复登录bug，第二是更新文档，第三是跟设计对一下新首页。",
       "收到，今天要处理三件事：\n1. 修复登录 bug\n2. 更新文档\n3. 跟设计对一下新首页", ["\n1.", "\n2.", "\n3."], ["嗯"], ["登录"], ["enumeration"]),
  case("sw-en-deploy", "en", "work", T("com.tinyspeck.slackmacgap"), "Quick update, the deploy is done, uh, but we still need to, um, rotate the API keys and update the dashboard.",
       "Quick update: the deploy is done, but we still need to rotate the API keys and update the dashboard.", ["API keys", "dashboard"], [" uh,", " um,"], ["deploy"]),
  case("sw-zh-short", "zh", "work", T("com.alibaba.DingTalkMac"), "好的老板，那个方案我下午三点前发您。",
       "好的老板，方案我下午三点前发您。", ["三点"], ["\n"], ["方案"]),
  case("sw-en-release", "en", "work", T(C, site="teams.microsoft.com"), "For the release we need two things, first finish the QA pass, second get sign off from legal.",
       "For the release we need two things:\n1. Finish the QA pass\n2. Get sign-off from legal", ["\n1.", "\n2."], [], ["QA", "legal"], ["enumeration"]),
  case("sw-zh-meeting", "zh", "work", T("com.tencent.WeWorkMac"), "各位，周会改到明天上午十点，呃，大家记得带上本周的数据。",
       "各位，周会改到明天上午十点，大家记得带上本周的数据。", ["明天上午十点"], ["呃"], ["本周的数据"]),
  case("sw-mx-review", "mixed", "work", T(C, site="app.slack.com"), "这个 PR 我 review 完了，有两个问题，一是 error handling 不完整，二是缺少 unit test。",
       "这个 PR 我 review 完了，有两个问题：\n1. error handling 不完整\n2. 缺少 unit test", ["\n1.", "\n2.", "error handling", "unit test"], [], ["PR"], ["enumeration"]),
  case("sw-zh-steps", "zh", "work", T(S, site="linear.app"), "这个问题，嗯，复现步骤是先登录，然后打开设置页，最后点保存就会崩溃。",
       "这个问题的复现步骤是先登录，然后打开设置页，最后点保存就会崩溃。", ["设置页", "保存"], ["嗯"], ["崩溃"]),
  # prompt
  case("sp-mx-retry", "mixed", "prompt", T("com.todesktop.230313mzl4w4u92"), "帮我改一下 src/api/client.ts，嗯，请求失败的时候要重试三次，然后，呃，超时设成五秒。",
       "帮我改一下 src/api/client.ts：\n1. 请求失败的时候要重试三次\n2. 超时设成五秒", ["src/api/client.ts", "\n1.", "\n2."], ["嗯", "呃"], ["三次", "五秒"], ["enumeration"]),
  case("sp-en-hook", "en", "prompt", T("com.anthropic.claudefordesktop"), "Refactor the useAuth hook so it, uh, returns the session and the loading state, and don't change the public API.",
       "Refactor the useAuth hook so it returns the session and the loading state, and don't change the public API.", ["useAuth"], [" uh,"], ["public API"]),
  case("sp-zh-app", "zh", "prompt", T(C, site="chatgpt.com"), "嗯，我想做一个记账的小程序，需要支持，第一，按月统计，第二，导出Excel，第三，多人共享账本。",
       "我想做一个记账的小程序，需要支持：\n1. 按月统计\n2. 导出 Excel\n3. 多人共享账本", ["\n1.", "\n2.", "\n3.", "Excel"], ["嗯"], ["记账"], ["enumeration"]),
  case("sp-en-tests", "en", "prompt", T("com.apple.Terminal"), "Run the tests in the payments package and, um, fix any failures but don't touch the migrations folder.",
       "Run the tests in the payments package and fix any failures, but don't touch the migrations folder.", ["payments", "migrations"], ["um,"], ["don't"]),
  case("sp-mx-issue", "mixed", "prompt", T(S, site="github.com"), "这个 bug 在 Safari 17 上能复现，呃，报错是 TypeError cannot read properties of undefined。",
       "这个 bug 在 Safari 17 上能复现，报错是 TypeError: cannot read properties of undefined。", ["Safari 17", "TypeError"], ["呃"], ["undefined"]),
  case("sp-zh-async", "zh", "prompt", T("com.microsoft.VSCode"), "把这个函数改成异步的，那个，用 async await，然后加上错误处理。",
       "把这个函数改成异步的，用 async/await，然后加上错误处理。", ["async", "await"], ["那个，"], ["错误处理"]),
  case("sp-en-backoff", "en", "prompt", T("dev.zed.Zed"), "Add a retry with exponential backoff, I mean linear backoff, to the fetchUser function.",
       "Add a retry with linear backoff to the fetchUser function.", ["linear backoff", "fetchUser"], ["exponential", "I mean"], [], ["self-correction"]),
  # search
  case("ss-zh-weather", "zh", "search", T(C, site="www.google.com"), "嗯，杭州明天天气怎么样。",
       "杭州明天天气怎么样", ["杭州明天天气"], ["。", "嗯"], []),
  case("ss-en-headphones", "en", "search", T(S, site="www.bing.com"), "best noise cancelling headphones under two hundred dollars.",
       "best noise cancelling headphones under two hundred dollars", ["noise cancelling headphones"], ["."], []),
  case("ss-zh-mailsearch", "zh", "search", T("com.apple.mail", field="search"), "上个月的报销单。",
       "上个月的报销单", ["上个月的报销单"], ["。"], []),
  case("ss-mx-python", "mixed", "search", T(C, site="www.baidu.com"), "呃，Python 怎么读取 Excel 文件？",
       "Python 怎么读取 Excel 文件", ["Python", "Excel"], ["呃", "？"], []),
  case("ss-en-reset", "en", "search", T(E, site="duckduckgo.com"), "um how to reset a mac to factory settings?",
       "how to reset a Mac to factory settings", ["factory settings"], ["um ", "?"], []),
  case("ss-zh-finder", "zh", "search", T("com.apple.finder", field="search"), "去年的发票。",
       "去年的发票", ["去年的发票"], ["。"], []),
  # chat
  case("sc-zh-dinner", "zh", "chat", T("com.tencent.xinWeChat"), "嗯，我晚上七点到，你们先吃吧。",
       "我晚上七点到，你们先吃吧。", ["七点"], ["嗯", "\n"], ["先吃"]),
  case("sc-en-saturday", "en", "chat", T("com.apple.MobileSMS"), "Hey, uh, are we still on for Saturday?",
       "Hey, are we still on for Saturday?", ["Saturday", "?"], [" uh,", "\n"], []),
  case("sc-zh-notes", "zh", "chat", T("com.apple.Notes"), "今天读完了第三章，呃，感觉作者对历史的分析很有意思。",
       "今天读完了第三章，感觉作者对历史的分析很有意思。", ["第三章"], ["呃", "\n"], ["历史"]),
  case("sc-zh-places", "zh", "chat", T("com.tdesktop.Telegram"), "周末想去两个地方，一个是西湖，一个是灵隐寺。",
       "周末想去两个地方，一个是西湖，一个是灵隐寺。", ["西湖", "灵隐寺"], ["\n"], [], ["enumeration"]),
  case("sc-en-wiki", "en", "chat", T(S, site="en.wikipedia.org"), "I think, um, this paragraph needs a citation.",
       "I think this paragraph needs a citation.", ["citation"], ["um,", "\n"], []),
  case("sc-mx-game", "mixed", "chat", T("com.hnc.Discord"), "今晚八点开黑，嗯，我先去买个 coffee。",
       "今晚八点开黑，我先去买个 coffee。", ["八点", "coffee"], ["嗯", "\n"], []),
]
corpus = {"version": 1, "kind": "arion-scene-text-v1", "split": "test", "license": "developer-authored synthetic",
          "consent": True, "annotation": "single developer; exact-string layout proxies; written before live-v3 demonstrations",
          "samples": samples}
data = json.dumps(corpus, ensure_ascii=False, indent=1)
open(sys.argv[1], "w").write(data + "\n")
print(len(samples), hashlib.sha256((data + "\n").encode()).hexdigest())
