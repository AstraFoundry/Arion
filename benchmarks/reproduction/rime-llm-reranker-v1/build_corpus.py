"""Freeze an authored exploratory corpus before collecting candidates/inference."""
import json
from pathlib import Path
import sys

cases = []


def add(group, context, pinyin, acceptable, purpose):
    cases.append({"id": f"{group}-{sum(c['group'] == group for c in cases) + 1:03d}",
                  "group": group, "context": context, "input": pinyin,
                  "acceptable": acceptable.split("/"), "purpose": purpose})


# Opposing contexts keep the phonetic input fixed and change the intended reading.
for line in """
beijing|这幅图的|背景|下周乘飞机去|北京
shiyan|化学课上做|实验|婚礼上许下|誓言
quanli|法律保障公民的|权利|为了赢下比赛拼尽|全力
gongshi|求解方程先写出|公式|录用名单需要公开|公示
sheji|请帮这款产品做视觉|设计|这起案件还会|涉及
shishi|下一步组织人员|实施|判断必须依据客观|事实
yiyi|这件事很有教育|意义|如果不同意可以提出|异议
zhiding|讨论后共同|制定|把文件放进|指定
yizhi|我们对方案的意见完全|一致|雨从昨天到现在|一直
shijie|祝愿各国人民生活在和平的|世界|又到了春暖花开的|时节
jicheng|子女依法可以|继承|这块芯片采用高度|集成
shengming|律师代表公司发表|声明|急救人员正在抢救伤员的|生命
shenqing|请提前提交书面|申请|她望着故乡时眼含|深情
yujing|台风来临前发布|预警|这个词要结合上下|语境
lianjie|点击网页上的蓝色|链接|请把两根电线|连接
jiaocheng|初学者需要一份入门|教程|山路很远非常考验|脚程
jiedian|图中的圆点表示网络|节点|离开房间随手关灯可以|节电
jiemian|这个软件的操作|界面|沿垂直方向切开观察|截面
gongcheng|桥梁正在施工这项|工程|多年努力终于|功成
kecheng|学校下学期新增一门|课程|守卫严密让敌人无机|可乘
qizhong|一共有五人参加|其中|学期过半即将举行|期中
zhili|城市污染需要共同|治理|这道题考验逻辑与|智力
shouyi|这笔投资产生稳定|收益|木匠靠祖传的|手艺
ziyuan|请节约宝贵的自然|资源|这次报名完全出于|自愿
""".strip().splitlines():
    pinyin, context_a, target_a, context_b, target_b = line.split("|")
    add("context", context_a, pinyin, target_a, "Opposing-context homophone selection")
    add("context", context_b, pinyin, target_b, "Opposing-context homophone selection")

for pinyin, target in [("nihao", "你好"), ("xiexie", "谢谢"), ("mingtian", "明天"),
                       ("pengyou", "朋友"), ("xuexi", "学习"), ("gongzuo", "工作"),
                       ("diannao", "电脑"), ("tianqi", "天气"), ("kuaile", "快乐"),
                       ("yinyue", "音乐"), ("lvxing", "旅行"), ("nvhai", "女孩")]:
    add("common", "", pinyin, target, "No-context familiar-word preservation")

for context, pinyin, target in [("这张图片需要换个", "bj", "背景"), ("出差第一站去", "bj", "北京"),
                                ("点击网页", "lj", "链接"), ("学校开设新", "kc", "课程"),
                                ("今天做化学", "sy", "实验"), ("法律保护公民", "ql", "权利"),
                                ("这是一个网络", "jd", "节点"), ("请填写入学", "sq", "申请")]:
    add("abbreviation", context, pinyin, target, "Initial-only Pinyin")

for context, pinyin, target in [("", "womenmingtianjian", "我们明天见"), ("", "qingguanshangchuanghu", "请关上窗户"),
                                ("", "jintiantianqihenhao", "今天天气很好"), ("", "wozhengzaixuexi", "我正在学习"),
                                ("", "qingfasongwenjian", "请发送文件"), ("", "huanyinglaidaozheli", "欢迎来到这里"),
                                ("", "zhegefangfahenhao", "这个方法很好"), ("", "qingbaocunwenjian", "请保存文件")]:
    add("long-input", context, pinyin, target, "Original preedit-length gate; do not relax it")

for context, pinyin, target in [("下周乘飞机去", "beij", "北京"), ("这幅图的", "beij", "背景"),
                                ("化学课上做", "shiy", "实验"), ("点击网页上的", "lianj", "链接"),
                                ("这款软件的", "jiem", "界面"), ("法律保障公民的", "quanl", "权利"),
                                ("提交书面", "shenq", "申请"), ("新的网络", "jied", "节点")]:
    add("partial", context, pinyin, target, "Incomplete final syllable; predictive target may be absent")

for pinyin, targets in [("ta", "他/她/它"), ("shi", "是/时/事/十/市/使/式/世/试/石/师/诗/食/史"),
                        ("de", "的/得/地"), ("yi", "一/以/已/意/义/亿/易/衣/医/依/艺")]:
    add("ambiguous", "", pinyin, targets, "Explicit multiple acceptable readings; no uniquely intended answer")

for context, pinyin, target in [("启动项目的", "oulayinqing", "欧拉引擎"), ("刚创建的分支叫", "lanhaixing", "蓝海星"),
                                ("这次测试使用", "xinghexieyi", "星河协议"), ("我们的产品叫", "alian", "阿里安")]:
    add("coined-name", context, pinyin, target, "Authored names; preserve oracle misses")

for prefix, pinyin, target in [("这幅图的", "beijing", "背景"), ("下周乘飞机去", "beijing", "北京"),
                               ("化学课上做", "shiyan", "实验"), ("婚礼上许下", "shiyan", "誓言"),
                               ("法律保障公民的", "quanli", "权利"), ("为了赢下比赛拼尽", "quanli", "全力"),
                               ("求解方程先写出", "gongshi", "公式"), ("录用名单需要公开", "gongshi", "公示")]:
    add("truncated-context", prefix + "（前面的事情已经说完，现在继续输入）" * 3, pinyin, target,
        "Relevant clue intentionally outside the upstream 60-byte context tail; diagnostic only")

assert len(cases) == 100
destination = Path(sys.argv[1])
with destination.open("x") as stream:
    json.dump({"version": 1, "id": "rime-llm-reranker-v1", "split": "authored-exploratory",
               "license": "AGPL-3.0-only (repository-authored fixtures)",
               "annotation": "Single assistant-authored targets, frozen before inference; no independent human review or unseen-population claim",
               "cases": cases}, stream, ensure_ascii=False, indent=2)
    stream.write("\n")
