"""Build an offline iteration report from pinned raw observations and explicit review."""
import base64
import importlib.util
import sys
import hashlib
import html
import json
import pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
OUT=ROOT/'benchmarks/results/rewrite-release'

ROUND_DEFINITIONS=[
 dict(id='alignment-v1',title='场景与语义审计起点',direction='比较原生产模板、场景模板与同模型语义审计；本次补充事后逐条复核。',finding='结构代理分上涨掩盖了翻译、完成态和回答请求等问题；未采用，原实验不改写。',scope='原始 96 条改写的新人工复核视图，没有新增推理。另有 113 条原审计结果单列。',runs=[('rewrite-retrospective-v1/dev','retrospective'),('rewrite-retrospective-v1/test','retrospective')]),
 dict(id='scene-v2',title='场景与语言示例',direction='固定语言、分离场景示例，允许工作内容按条目组织。',finding='新集的场景适配改善，但忠实度下降；不采用。',scope='开发 16 条、旧集 24 条、新集 32 条，分别配对；下方展示全部实际生成。',runs=[('scene-rewrite-prompt-v2/dev','main'),('scene-rewrite-prompt-v2/evaluation','main')]),
 dict(id='structure-v3',title='自然结构化',direction='从用户认可的案例提炼负责人／任务／日期归组、邮件称呼与正文分段。',finding='小规模开发探针有效，含两个已进入示例的锚点；不作为泛化证明。',scope='8 条开发输入 × 2 版本；不含 OCR 纠错验收。',runs=[('scene-structure-v3','direct')]),
 dict(id='context-v1',title='OCR 支持的纠错',direction='允许相关 OCR 修复人名、术语、函数名，同时加入错 OCR 与无关信息对照。',finding='能纠正部分姓名与标识符，但错误 OCR 仍可覆盖原文。',scope='24 条人工构造文本 × 3 版本，非实际 OCR／ASR 全链路。',runs=[('context-repair-v1/development','main')]),
 dict(id='ungated-v1',title='内容保真与取消校验',direction='补充约束、否定和请求语气示例，重放 strict／envelope／none 的最终交付。',finding='新候选组织更好，完全取消文本校验会放出漏约束和协议内容。',scope='16 条开发输入 × 2，加 24 条旧集和 24 条新输入 × 2；基线是上一候选组合。',runs=[('rewrite-no-validator-v1/dev','main'),('rewrite-no-validator-v1/evaluation','main')]),
 dict(id='ocr-v2',title='拼写对齐规则',direction='明确拼写不同不是事实冲突，复制完整标识符，并保留日期、条件和否定。',finding='纠错正例增加，但旧集出现更多上下文替代口述；尚不能上线。',scope='16 条开发 × 2，加 24 条旧集和 24 条新输入 × 2；另有 12 次复现对照。',runs=[('ocr-alignment-v2/dev','main'),('ocr-alignment-v2/evaluation','main'),('ocr-alignment-v2/parity','parity')]),
 dict(id='packaging-v1',title='输入分区封装',direction='保持模型、模板、示例和预算一致，只比较 XML／JSON／明确分区。',finding='明确分区减少上下文替代，但纠错漏改与组织不足仍存在；仅作为下一轮开发基线。',scope='18 条开发 × 3；选择后旧 24 与新 24 × 2，150 条全部 review。',runs=[('rewrite-packaging-v1/dev','main'),('rewrite-packaging-v1/evaluation','main')]),
 dict(id='isolation-v1',title='干净示例与规则语言',direction='保留分区，移除示例中的攻击字符串，并比较中英文编辑规则。攻击仍保留在实际输入。',finding='两个候选均退步，保留基线；未执行新集测试。',scope='18 条开发 × 3，54 条全部 review；不把未跑新集计为成功。',runs=[('rewrite-prompt-isolation-v1/dev','main')]),
 dict(id='task-wrapper-v1',title='当前输入的场景任务',direction='在正文前加入明确场景任务，保持模型、系统模板与示例一致。',finding='列表更稳定，但疑问变判断，且耗时增加；不采用。',scope='18 条开发 × 2；旧 24 与此前未生成的新 24 × 2，132 条全部 review。',runs=[('rewrite-task-wrapper-v1/dev','main'),('rewrite-task-wrapper-v1/evaluation','main')]),
 dict(id='entity-evidence-v1',title='实体拼写证据与实际 OCR',direction='把拼写规则改成“先精确匹配，再唯一同音匹配”，并实际调用 macOS Vision 的 accurate／fast。',finding='新规则未改善；错误 OCR 仍能覆盖正确原文。示例中的名字在无 OCR 时也会被复述，不能当成 OCR 收益。',scope='26 条开发 × 2；12 张固定文字图 × 2 OCR 模式 × 2 提示词，共 100 条改写和 24 条实际 OCR。',runs=[('rewrite-entity-evidence-v1/dev','main'),('rewrite-entity-evidence-v1/vision','main')]),
 dict(id='model-screen-v1',title='本地模型与导出对照',direction='保持提示词、分区、预算与运行时一致，对比已验证字节的 1.7B 与现有 4B 导出。',finding='4B 改善归组和人名，但引入其他原意损失且更慢；不采用。',scope='同 8 条已暴露开发输入 × 2 模型，共 16 条。两模型的检查点与量化导出不同，不能归因于参数量一个因素。',runs=[('rewrite-model-screen-v1/development','main')]),
 dict(id='4b-packaging-v1',title='4B 的输入封装',direction='同一 4B 模型与系统模板，比较明确分区、JSON、XML。',finding='JSON 保住了疑问与完成态，但漏人名纠错；没有格式通过开发门槛，不消耗新集。',scope='8 条开发 × 3 封装，24 条全部 review；预先冻结的 16 条新输入未生成。',runs=[('rewrite-4b-packaging-v1/dev','main')]),
 dict(id='4b-entity-v1',title='4B JSON 的拼写规则',direction='固定 JSON 与 4B，仅将实体规则改成先精确匹配、再唯一同音匹配。',finding='仍漏改人名，并重新出现 OCR 替代原文；不采用。',scope='8 条开发 × 2 规则，共 16 条；新集未执行，不声称泛化或上线通过。',runs=[('rewrite-4b-entity-v1/dev','main')]),
 dict(id='small-models-v1',title='最新一代 ≤2B 文本模型',direction='停止4B与旧代Qwen测试；固定CPU4线程、非思考、提示词与分区，比较Qwen3.5-0.8B和2B文本GGUF。',finding='完整呈现无校验器过滤的输出；成功与失败均计入，尚未采用。',scope='24条固定输入×2：8条已暴露探针、16条此前未生成探针；纯文本张量约0.75B／1.88B，不加载视觉编码器。新运行时、禁用前缀缓存，不与历史ONNX时间作模型单因素比较。',runs=[('rewrite-small-models-v1/qwen3.5-0.8b','gguf'),('rewrite-small-models-v1/qwen3.5-2b','gguf')]),
 dict(id='small-prompt-v1',title='0.8B 的四场景提示词封装',direction='明确中文编辑边界并替换工作示例，检查负责人归组、唯一拼写证据、条件和限制。',finding='修复一条 OCR 替代，却漏改原本能修复的函数名；归组和歧义仍失败。开发门槛未过，不采用。',scope='16条已暴露开发×2封装，共32条；四场景各4条。预先冻结24条新输入，未生成。',runs=[('rewrite-small-prompt-v1/dev','gguf')]),
 dict(id='small-cache-v1',title='0.8B 的原生前缀缓存',direction='保持模型、模板与解码不变，仅比较缓存开关；按正反两种顺序复验。',finding='确实命中缓存且更快，但2/16组原始文本不同；意思保住了，仍不满足逐字一致性要求。',scope='8条已暴露工作／邮件探针×2缓存×2顺序=32次正式请求，另8次预热，40条全部review。非完整质量或原生交付验收。',runs=[('rewrite-small-cache-v1/run','gguf')]),
 dict(id='small-cache-failed-v1',title='单 token 逻辑批：启动失败',direction='尝试统一缓存前后数值计算路径，设置逻辑批与物理微批均为1。',finding='原生能力探针需要2个token，逻辑容量1使运行时崩溃；保留失败，不计作推理通过。',scope='0次生成；失败清单、原始日志、实际环境和对应版本的源码诊断均保留。',runs=[]),
 dict(id='small-cache-parity-v2',title='单 token 微批的一致性复验',direction='根据启动日志和源码保留默认逻辑容量，只把物理微批设为1；其余处理不变。',finding='16/16缓存对照、16/16顺序对照逐字一致，但缓存开启p95约1.61秒且两轮耗时漂移明显，未采用为上线配置。',scope='同8条探针×2缓存×2顺序=32次正式请求，另8次预热，40条全部review。无真实ASR并发／停止至写入测量。',runs=[('rewrite-small-cache-parity-v2/run','gguf')]),
 dict(id='2b-contrast-v1',title='2B 的同音歧义对照提示',direction='只改工作场景，明确比较全部同音候选，加入唯一与歧义成对示例。',finding='两版仍擅自选择readCash；负责人归组和邮件成功案例保留，未通过开发门槛。',scope='16条暴露开发×2，共32条。12条非工作请求与输出逐字相同，但耗时漂移；新24条未生成。',runs=[('rewrite-2b-contrast-v1/dev','gguf')]),
 dict(id='pronunciation-evidence-v1',title='显式发音候选输入证据',direction='固定两臂规则，只给候选注入CMU词典计算的唯一／歧义／未知匹配；保留完整ASR与OCR。',finding='补回一条buildCache纠错，但三条同音歧义仍被自由生成忽略；不采用。',scope='24条暴露开发×2，共48条全部review。文本推导不等于声学证据；仅覆盖部分英文复合词，不覆盖中文姓名及未知词的完整候选。新24条未生成。',runs=[('rewrite-pronunciation-evidence-v1/dev','gguf')]),
 dict(id='source-plan-v1',title='语义片段与受约束编辑方案',direction='为每个原文片段保留必需方案项，对同音歧义设置字面锚点，允许其余部分润色并由方案构造正文。',finding='歧义字面保留改善，但必要纠错全部漏改，工作归组和邮件分段退步；这版方案不采用。',scope='28条暴露开发×2，共56次生成；包含中文纠错、口语润色与自我修正。384token预算一致，协议失败留在分母；新24条未生成。',runs=[('rewrite-source-plan-v1/dev','sourceplan')]),
 dict(id='local-protection-v1',title='局部词保护与非Qwen小模型',direction='修正多要求组织示例，补入口语润色；仅对明确同音歧义做可逆词编码，保留正文自由改写，并比较Granite1.46B。',finding='Qwen主集保真改善并保住全部必要纠错；Granite漏日期、空输出且服从恶意OCR。补充压力集失败，均未采用。',scope='28条暴露开发×Qwen直接／Qwen保护／Granite直接，共84次生成；必要纠错9条。仅编码协议检查，无语义过滤或源文回退；新24条未生成。',runs=[('rewrite-local-protection-v1/qwen-direct','protected'),('rewrite-local-protection-v1/qwen-protected','protected'),('rewrite-local-protection-v1/granite-direct','protected')]),
 dict(id='local-protection-stress-v1',title='局部保护的泛化压力测试',direction='补测多词歧义、自我纠正、重复提及、引号原样要求、条件语气与误导／恶意OCR，避免只看已暴露主集。',finding='局部保护保住部分条件与疑问，但多词仍会猜、引号被改、现金／缓存语义被OCR替代，重复任务没有去重。不能上线。',scope='8条新增人工压力输入×Qwen直接／保护，共16条全部review。2次编码拒绝的原始文本也有错误，本次未观察到正确正文被协议误拒。新24条仍未生成。',runs=[('rewrite-local-protection-stress-v1/qwen-direct','protected'),('rewrite-local-protection-stress-v1/qwen-protected','protected')]),

 dict(id='current-task-v1',title='当前任务尾部与取消编码拒绝',direction='保留全部36条回归，比较原保护版、正文尾部任务、允许自然省略且取消输出拒绝。',finding='尾部规则被误当成口述正文复制；取消拒绝未修复语义错误，候选均不采用。',scope='36条已暴露输入×3，共108条全部逐条review。保留原始与还原正文，11条必要纠错。新24条未生成。',runs=[('rewrite-current-task-v1/qwen-protected','protected'),('rewrite-current-task-v1/qwen-recency','protected'),('rewrite-current-task-v1/qwen-flexible','protected')]),
 dict(id='system-task-probe-v1',title='系统层任务边界修正',direction='将同义编辑规则从用户正文尾部移入system层，明确最后一条用户消息中的待改写数据位置；继续不做输出拒绝。',finding='修复4条尾部封装退化，但难例仅8/12忠实、7/12场景可用，与原保护版同一子集持平；仍不采用。',scope='12条已暴露难例可行性探针，非完整36条回归。失败后未扩大36条或新24条；不缩小最终验收范围。',runs=[('rewrite-system-task-probe-v1/dev','protected')]),

 dict(id='private-spans-v1',title='实际口述清理与私有词映射',direction='两臂对齐实际CleanSpeech；保护引用内容、重复词共用标记、隐藏还原映射和重叠发音候选，保留完整OCR与自然正文。',finding='多词、引用、去重与必要纠错改善，但新增无关OCR函数正文；另一条定义含义保留、readCache拼写替换仍无依据。仍未采用。',scope='完整36条已暴露回归×2，共72条全部逐条review。两臂均不做输出拒绝或原文回退；实际清理输入与原ASR分别保留，新24条未生成。',runs=[('rewrite-private-spans-v1/qwen-clean-system','protected'),('rewrite-private-spans-v1/qwen-private-spans','protected')]),

 dict(id='data-object-v1',title='参考字段与统一数据对象',direction='复用原私有映射基线，分别只改参考字段大小写，或把全部示例／当前用户输入统一为JSON并以system指定dictation字段。',finding='小写版未修复无关OCR追加且定义再次反转；JSON版修复追加，却漏姓名纠错、替换多词和条件目标。均不采用。',scope='完整36条已暴露回归×2，新增72条全部逐条review；旧基线36条只作对照、不重复计入推理。完整OCR与实际清理均保留，无输出拒绝；新24条未生成。',runs=[('rewrite-data-object-v1/qwen-lowercase','protected'),('rewrite-data-object-v1/qwen-data-object','protected')]),

 dict(id='2b-precision-v1',title='同一2B导出族的Q8对照',direction='保持36条请求、私有映射、清理与解码完全一致，只换同仓库同修订Q8_0导出；复用Q4原始基线。',finding='修复无关OCR追加，但条件目标替换、姓名和buildCache漏纠、现金／缓存定义反转仍出现；不采用。',scope='新增36条Q8原生生成全部逐条review，旧Q4基线不重复计数。实际1.88B参数，权重约2.01GB；没有新增4B或旧Qwen推理。未取得16GB或上线验收。',runs=[('rewrite-2b-precision-v1/qwen-q8-private','protected')]),
 dict(id='bounded-edit-v1',title='有界拼写选择与独立排版',direction='完整OCR进入有界拼写选择；只将纠正后的口述交给自然排版，保持可逆词保护、自由组织和无输出拒绝。',finding='消除一类OCR事实注入，但漏纠错、删除收件人、反转限制与否定仍发生；不采用。',scope='36条暴露回归输入×选择／排版两阶段=72次真实生成；仅36条最终正文。新24条未生成。架构研究，尚未用于生产。',runs=[('rewrite-bounded-edit-v1/run','staged')]),
 dict(id='bounded-edit-v2',title='同字母规范化与可见纠正原文',direction='采用用户确认的拼写口径、英文词窗与正向纠错示例；排版阶段直接看纠正原文，保留自由结构化和无过滤输出。',finding='必要拼写纠错和收件人保留改善；新增查记录的范围变化、删除明确排除说明，仍未通过完整门槛。',scope='完整36条已暴露回归×选择／排版=72次真实生成，36条最终正文；与v1整套方案对比，不是单项提示词因果实验。新24条未生成。',runs=[('rewrite-bounded-edit-v2/run','staged')]),
 dict(id='formatter-v3',title='排版的范围、强调与口头填充',direction='复用v2实际纠错结果，只调整排版提示词：禁止补检查细节、区分改口和数字对比、清理多余语气词。',finding='移除了新增查记录细节，聊天更自然；四秒数值保留，但仍省略不是四十秒的强调。按冻结的完整强调口径未通过；该偏好已单独询问。',scope='36条暴露回归，新增36次最终排版生成；复用36条真实选择及36条v2正文，均不重复计数。非连续两阶段计时；新24条未生成。',runs=[('rewrite-formatter-v3/run','staged')]),
 dict(id='metal-preflight-v1',title='macOS Metal 可运行性探针',direction='复用同一2B权重与同一条排版请求，仅请求Metal层；另用无生成的详细加载观察实际放置。',finding='一次正文保住负责人任务日期，但与CPU措辞不同；同配置加载观察25/25层放置于GPU。尚不能认定质量、延迟或16GB验收通过。',scope='新增1次真实无上下文排版生成，另一次仅加载0生成。请求约1542ms不是p95或两阶段／ASR并发停止至写入指标。',runs=[('rewrite-metal-preflight-v1/run','staged')]),
 dict(id='metal-regression-v1',title='Metal 完整36条排版回归',direction='保持实际请求、模型、预算及纠错结果一致，完整比较CPU与Metal，并在每个进程启动时记录GPU放置。',finding='35/36按冻结强调口径通过，11/11纠错保留；34/36输出逐字相同。单排版阶段仍不等于完整链路。',scope='36次新生成，复用历史36选择与CPU正文不重复计数；M4/32GB，非16GB或ASR并发。',runs=[('rewrite-metal-regression-v1/run','staged')]),
 dict(id='natural-format-v1',title='120条自然转写首次候选验证',direction='冻结同一提示词与Metal，输入历史中文／英文ASR各60条；不传参考转录，不要求凭空恢复缺失内容。',finding='出现翻译、人物视角和动作／请求／细节改变；源文原样保留也不一定自然可用。3条不确定判断单列。',scope='120次新生成，零新增ASR／OCR／选择模型调用。历史朗读，非新留出、真实场景或独立盲评。',runs=[('rewrite-natural-format-v1/run','staged')]),
 dict(id='language-profile-v1',title='语系证据与英文编辑权限',direction='用原文字符计算Han/Latin存在情况，改写通用及场景指令为英文，补充独立语法／代词／残句示例。',finding='36条场景门槛通过，自然文本多数问题改善；仍有一条英译中，新增一条中译英，未采用。',scope='36场景＋120自然文本，共156次真实生成；全部原文与纠错结果固定，零新增选择或ASR调用。',runs=[('rewrite-language-profile-v1/run','staged')]),
 dict(id='language-profile-v2',title='按原语系封装指令与示例',direction='纯汉字源文使用中文规则及纯中文示例，纯拉丁文字使用英文规则及无汉字示例，混输保留多语示例；不裁剪原文或参考上下文。',finding='本组语言转换消失，保真改善；仍有1条含糊补词待核验，语法／源文残缺影响可用性。中文改口仍保留口头过程，单独记作排版退步。',scope='完整156条配对新生成；无输出过滤或原文回退，未通过这些暴露数据宣布上线。',runs=[('rewrite-language-profile-v2/run','staged')]),
 dict(id='cleanup-copula-v1',title='明确改口的确定性清理修复',direction='修复“不对，是四秒”因“是”挡住数量匹配而漏清理。仅完整同类型数量替换可删除旧值／提示／系词，保留否定与其他约束。',finding='实际Go执行156源文：新增清理1条，既有改口1条；其余不改。完整核心测试及模糊测试通过；不改写此前实验输入。',scope='零LLM／ASR生成。speech-spans-v2源码修复，非新模型或原生完整链路上线验收。',runs=[]),
 dict(id='metal-chain-v1',title='实际清理、按需纠错与排版全流程',direction='逐条执行当前Go清理；有候选才实际调用Metal拼写选择，然后运行原语系排版。统一跳过零候选，不复用旧模型选择。',finding='36条场景保真／可用与11条必要纠错通过；自然119保真、1待核验、75可用。场景文本链路p95约2.92秒，仍未达到延迟门槛。',scope='新增174次模型生成：18次实际选择＋156条最终正文；138条零候选直接排版。当前M4/32GB，不含ASR、真实OCR、原生写入或16GB验收。',runs=[('rewrite-metal-chain-v1/run','staged')]),
 dict(id='metal-prefix-v1',title='完整文本链路的原生前缀复用',direction='保持174条请求的全部内容、模型与运行设置，仅开启cache_prompt；逐条对照未缓存原始输出，保留首次缓存未建立的请求。',finding='174/174原始输出逐字一致，164次命中；场景p50约694ms、p95约2420ms，自然p95约685ms。首次场景／语系切换仍慢，未达完整延迟验收。',scope='新增174次实际推理，全量review；同156源文／前向顺序，无预热。基线是历史串行观察，非随机交错计时或隐私／16GB验收。',runs=[('rewrite-metal-prefix-v1/run','staged')]),
 dict(id='metal-primed-v1',title='通用预热、输入顺序与上下文标记',direction='每个场景进程预热拼写选择与三种原语系排版前缀；全部156输入按正逆顺序复测，保留所有预热成本与标记控制。',finding='348条正式输出均匹配原基线，跨顺序174/174逐字一致；每组12个正向标记均保留，156条正文无标记串入。就绪后场景p95正序836ms、逆序1470ms；仍非原生／并发／16GB验收。',scope='380次实际生成：348正式＋32预热，全部review；每场景进程加载及预热4.77–8.15秒单列，未当作免费。原36＋120来源重复，不是新增独立样本。',runs=[('rewrite-metal-primed-v1/forward','staged'),('rewrite-metal-primed-v1/reverse','staged')]),

 dict(id='history-chain-v1',title='完整历史输入回归与漏纠错定位',direction='对历史401个来源／完整上下文／指定场景元组执行真实清理、按需拼写选择和排版，保留全部输出。',finding='401正文中397确认保真、2未满足保留口径、2待核验，335可用；51例支持的拼写编辑保留，7个漏规范机会中5个未产出候选、2个模型选择保留。',scope='465次实际生成：64次选择＋401正文；人工构造281／历史自然120分列。未预热，本轮用于质量覆盖，不与预热延迟直接比较；生产未启用。',runs=[('rewrite-history-chain-v1/run','staged')]),
 dict(id='history-format-v2',title='共同条件、请求与显式对比的提示词修订',direction='固定401源文、实际清理和64次旧拼写选择，仅修订四场景规则与非测试示例，重新生成全部401正文。',finding='确认保真397→399，可用335→342；共同时间、邮件请求和显式数值对比改善。新增英文背景事实遗漏与口头重复退化，未采用。',scope='401次新排版生成，全量review；构造281／自然120分列。未新增选择／清理，不计作全链路计时；保持全部原始输出，无校验过滤。',runs=[('rewrite-history-format-v2/run','staged')]),
 dict(id='work-scope-v3',title='完整工作消息与背景状态',direction='强调背景／状态和专属／共同条件分离；完整94条历史工作源，加16条新构造对照的双臂实际生成。',finding='历史仍为93/94保真、85可用：补回接口背景却漏掉另一条负责人不变；16条新对照双臂均保真／可用，不采用整版。',scope='126次实际生成：94历史候选＋16新对照×2，全部review。历史基线复用；对照未执行清理或选择，其他307场景未生成。',runs=[('rewrite-work-scope-v3/run','staged')]),
 dict(id='work-scope-v4',title='简化完整性规则与日期归组对照',direction='缩短工作指令，明确不变事项，将共同／专属日期对照示例重新排序，重复全部94历史工作源和16构造对照。',finding='历史94/94确认保真、87可用；保留背景与负责人不变，清理口头重复，恢复小陈／小刘的日期归组。16对照均保真／可用；7条未解术语及部分日期分离仍在。',scope='110次新候选正文全部review；旧v3响应只作对照，不重复计数。其他307场景仅验证请求逐字一致，零新生成；非完整链路或上线验收。',runs=[('rewrite-work-scope-v4/run','staged')]),
 dict(id='tone-chain-v1',title='声调无关拼写候选与完整链路回归',direction='保持选择提示词与格式器一致，仅放宽中文声调候选；候选实际跑完整401历史源＋24新对照，24对照另跑严格声调基线。',finding='历史401正文与已复核对应输出全部一致：400确认、1源文待核验、344可用。新对照纠错2/8→3/8，却把苗圃培植误改为配置；保真24→23/24、可用均16，不采用。',scope='538次实际响应：89选择＋449正文；449次真实Go清理，24对照双臂重复。无语义过滤或源文回退，非实际ASR／Vision／原生写入或16GB验收。',runs=[('rewrite-tone-chain-v1/run','staged')]),
 dict(id='selector-semantic-v1',title='拼写选择的语义角色与错误OCR对照',direction='候选与格式器固定，只调整选择提示词／示例；完整401历史＋24旧对照回归，新增32语义对照双臂。',finding='新对照误改6→2/32，但相关纠错仍0/4；旧纠错3→1/8，历史新增9条姓名漏纠、修复3条术语。历史可用344→338，拒绝采用。',scope='609次实际响应：120选择＋489正文，全量review；101候选选择行沿用旧模板元数据，原始记录保留，实际新请求与冻结提示词逐条绑定。开发诊断，非上线验收。',runs=[('rewrite-selector-semantic-v1/run','staged')]),
 dict(id='selector-semantic-v2',title='恢复实体纠错权限的简化选择提示',direction='保留相同候选与格式器，缩短语义规则并加入无关的正反角色对照，完整401历史＋56对照实际跑当前链路。',finding='相比上一版：旧纠错1→7/8，恢复9个姓名，历史可用338→346/401；仍有3个错误OCR替换、4个相关上下文正例漏纠，拒绝采用。',scope='558次新响应：101选择＋457正文，全部review；旧双臂基线只复用不重计推理。新执行器逐行记录实际模板路径／哈希／消息哈希，无语义拒绝或源文回退。开发重复数据，非上线验收。',runs=[('rewrite-selector-semantic-v2/run','staged')]),
 dict(id='selector-preview-v1',title='把候选替换结果作为完整句子展示',direction='保持规则／示例答案／候选／格式器不变，仅在选择数据中加入逐候选替换后的完整原句；完整457源实际跑当前链路。',finding='正例改善但严重退化：旧纠错7→8/8、相关语义纠错0→3/4，语义对照误改3→10/32，旧对照新增书架→暑假、合法→喝法。拒绝采用。',scope='558次新响应：101选择＋457正文，全部review；模型与运行时相同，260文件预冻结。去除预览字段后实际选择请求逐条匹配冻结基线；无语义过滤或源文回退。',runs=[('rewrite-selector-preview-v1/run','staged')]),
 dict(id='selector-screen-v1',title='成对候选展示与专名规则的选择阶段对照',direction='四组实际对照：原规则、单向预览、成对展示、成对展示加专名／普通词规则。先准备全部457源，逐组执行全部101个候选请求。',finding='符合要求92／88／94／83（各101），有害选择3／12／4／17。成对展示优于单向预览，但仍4误改3漏纠；增加规则严重退化，全部不通过。',scope='404次真实选择，全部review；457次实际Go清理在四组内复用，1828个源处理结果中1424次无候选不调用模型。零新增排版／交付正文，不能据此判断完整链路或上线。',runs=[('rewrite-selector-screen-v1/run','staged')]),
 dict(id='granite-runtime-v1',title='另一款1.63B模型的原生运行故障对照',direction='固定官方Granite4.0-1b与同一请求，比较Metal／CPU层放置、语法开关、KV精度和Flash Attention。',finding='Metal无语法约束时连续输出8；CPU层路径返回有效JSON。f32KV及关闭Flash均未修复Metal，不能归因为模型质量或已证明的NaN。',scope='首次202次计划在首个HTTP500后停止，完成0；另12次诊断尝试完成7个响应、5次HTTP错误，全部保留。7个响应单列为运行诊断，零交付正文。',runs=[('rewrite-selector-model-v1/diagnostic-responses','runtime')]),
 dict(id='selector-model-cpu-v1',title='Granite1.63B在相同纠错请求上的CPU层诊断',direction='固定官方Granite4.0-1b Q4，显式请求GPU层0，完整执行原规则／成对展示两组各101个候选请求。复用Qwen实际请求作对照。',finding='原规则／成对展示符合84／86（各101），有害替换16／13、漏纠1／2。相对既有Qwen对照30个选择改变：7改善、23退化；两组均不采用。',scope='202次新选择全部review、914个源处理结果含712次无候选跳过；全部457源准备显式复用旧实际Go结果，零新Go清理、排版或交付正文。Qwen对照未重计生成，provider也不同，不能作单因素模型因果结论。',runs=[('rewrite-selector-model-cpu-v1/run','staged')]),
 dict(id='selector-referent-v1',title='简短指代说明与有界拼写选择',direction='四组相同256token预算：原提示、含义核对、成对展示、成对展示加含义核对。保持全部457源与候选不变，实际执行404次选择，分别审查选择与说明。',finding='选择符合92／92／94／93（各101），有害3／4／4／5；说明没有改善选择。212条候选说明中44条未通过，其中27条选择正确但说明不准确或漏比对象。两版含义核对均不采用。',scope='404次新选择＋212条生成的候选说明，全部review；457次实际Go清理在四组内复用，1828源处理结果含1424无候选跳过。零新增正文／ASR／Vision，无语义过滤或原文回退。',runs=[('rewrite-selector-referent-v1/run','staged')]),
 dict(id='selector-source-last-v1',title='原文置于上下文末尾的字段顺序对照',direction='四组128token同预算：原规则、原文末尾、成对展示、成对展示加原文末尾。只移动十个示例和当前输入的顶层source字段，保留完整OCR、系统提示和示例答案。',finding='符合92／93／94／88（各101），有害3／5／4／12；原文末尾减少漏纠，却增加错误对象替换，配合成对展示退化更明显。两版均不采用。',scope='404次实际选择全部review；457次实际Go清理四组复用，1828个处理结果含1424无候选跳过。零新增正文／ASR／Vision，无语义过滤或原文回退。',runs=[('rewrite-selector-source-last-v1/run','staged')]),
 {'id': 'selector-sentence-choice-v1', 'title': '整句A/B选择与选项顺序对照', 'direction': '四组同预算：原规则、成对展示、整句A/B、整句B/A。新提示要求先比较原句对象与动作，再选择原句或纠正拼写后的整句；既有示例语义答案不变。', 'finding': '符合92／94／81／91（各101），有害3／4／19／5。A/B交换后归一选择一致77/101；完整结果未通过既定选择门槛，不进入正文或上线验收。', 'scope': '404次实际选择全部review；457次实际Go清理四组复用，1828个结果含1424无候选跳过。每组12个歧义病例沿用只能保留原句的schema，不当作自由语义选择成功。零新增正文／ASR／Vision。', 'runs': [('rewrite-selector-sentence-choice-v1/run', 'staged')]},
 {'id': 'hybrid-preflight-v1', 'title': 'Granite混合架构的原生推理与缓存对照', 'direction': '实际1.46B混合架构，Metal99与CPU层0各六次：相同完整请求、跨源返回、缓存开关及去除schema。核对实际模板和GPU放置。', 'finding': '12次均为有效JSON且正常结束，缓存开关三次原始输出各自一致。8次匹配诊断任务，4次仍把明确的现金读取误改为readCache；运行正常不代表内容正确。', 'scope': '12次真实运行探针，复用两个源，不能当作12个独立质量样本。Metal实际41/41GPU层，CPU层0为明确诊断对照；无新增正文／ASR／Vision。', 'runs': [('rewrite-selector-hybrid-v1/runtime-view', 'runtime')]},
 {'id': 'selector-hybrid-v1', 'title': '1.46B Granite混合架构的相同请求比较', 'direction': '保持原规则／成对展示两组完整请求和候选不变，替换为另一种2B以内模型与分词器。复用457个实际Go准备结果，执行全部202个候选请求。', 'finding': '符合81／82（各101），有害20／19；Qwen同请求对照为92／94、有害3／4。必需纠错均47/47，但错误OCR覆盖原意更多。两版均不采用。', 'scope': '202次真实选择全部review；914个源处理结果含712无候选跳过，零新增清理／正文／ASR／Vision。41个变化：9改善、32退化；权重与分词器／架构共同变化，不能归因于参数量。', 'runs': [('rewrite-selector-hybrid-v1/run', 'staged')]},
 {'id': 'instruction-language-qwen-v1', 'title': 'Qwen系统规则英译的单字段对照', 'direction': '只把系统规则从中文译成英文；中文示例、原文、完整OCR、候选、schema及采样不变。当前1.88B Qwen两组各101实际请求。', 'finding': '符合88／87，有害10／12、漏纠3／2；中文对照92／94，有害3／4。19个变化中4改善15退化；两版英译均不采用。', 'scope': '202次真实选择全部review；复用457个实际Go准备结果，914个处理结果含712无候选跳过。原有12个原句唯一选项的歧义病例每组保留，不算自主语义选择。零新增正文／ASR／Vision。', 'runs': [('rewrite-selector-instruction-language-v1/qwen', 'staged')]},
 {'id': 'instruction-language-hybrid-v1', 'title': '混合架构系统规则英译的单字段对照', 'direction': '同一份逐条对齐的英文规则，保留所有中文数据与示例，在当前1.46B混合架构上独立比较原规则／成对展示。', 'finding': '符合80／79，有害21／22；中文对照81／82，有害20／19。仅4个选择变化，全部退化；英文规则不能仅凭英语指令训练背景就采用。', 'scope': '202次真实选择全部review；完整请求还原中文系统消息后与既有对照逐项一致。复用准备结果，无新增清理／正文／ASR／Vision，无语义输出拒绝或原文回退。', 'runs': [('rewrite-selector-instruction-language-v1/hybrid', 'staged')]},
 {'id': 'selector-reference-turn-v1', 'title': '完整OCR分离为参考消息的封装对照', 'direction': '四组128token同预算；把十个示例和当前输入的完整context移至前置tool消息，保留原文、全部OCR、候选、系统规则及示例答案。实际模板将tool渲染成带tool_response标记的user回合，不代表权限隔离。', 'finding': '符合92／89／94／91（各101），有害3／9／4／8、漏纠6／3／3／2。分离参考回合未达到零有害且完成必要纠错的开发门槛，两版均不采用。', 'scope': '404次真实选择全部review；457次实际Go准备四组复用，1828个处理结果含1424无候选跳过。4次模板预检零生成，无新增正文／ASR／Vision，无语义过滤或原文回退。', 'runs': [('rewrite-selector-reference-turn-v1/run', 'staged')]},
 {'id': 'selector-unique-label-v1', 'title': '唯一候选标签是否诱导误改的单字段对照', 'direction': '四组128token同预算；只移除十个示例和当前输入里候选的status:unique。保留ambiguous标签、类型、全部拼写、完整OCR、系统规则、示例答案及schema；不改变实际候选生产器。', 'finding': '符合92／91／94／91（各101），有害3／4／4／7、漏纠6／6／3／3。仅移除unique标签仍未通过开发门槛，两版均不采用。', 'scope': '404次真实选择全部review；457次实际Go准备四组复用，1828个处理结果含1424无候选跳过。零新增正文／ASR／Vision，无语义过滤或原文回退。', 'runs': [('rewrite-selector-unique-label-v1/run', 'staged')]},
 {'id': 'selector-contrast-demo-v1', 'title': '新增语义正反例与逐候选判断的对照', 'direction': '四组128token同预算；原有十个示例后新增衣物缝线／项目风险、河堤护坡／琥珀饰品四例。正例同时示范纠正缝线→风险、保留先评估，完整保留原规则、原示例与当前OCR。', 'finding': '符合92／90／94／92（各101），有害3／3／4／6、漏纠6／8／3／3。两版新增语义示例均未通过，不采用。', 'scope': '404次真实选择全部review；457次实际Go准备四组复用，1828个处理结果含1424无候选跳过。808次本地模板／分词预检零推理；零新增正文／ASR／Vision，无语义过滤或原文回退。', 'runs': [('rewrite-selector-contrast-demo-v1/run', 'staged')]},
 {'id': 'minicpm-preflight-v1', 'title': 'MiniCPM5-1B原生模板与缓存探针', 'direction': '实际1.08B、Q4_K_M；Metal99与CPU层0各六次，保留完整原文与OCR，比较缓存关／开／关和去除schema。', 'finding': '12次均正常停止、JSON有效、缓存原始输出一致；12次语义判断均失败，包含漏纠loadCache和违背现金定义改成readCache。兼容性通过不代表质量通过。', 'scope': '12个完整响应全部review；另1个意外启动后中断的完成仅见原生日志，单独公开，不伪造响应或计入完整用例。模板末尾LF差异已核对固定版本Jinja源码。零最终正文／ASR／Vision。', 'runs': [('rewrite-selector-minicpm-v1/runtime-view', 'runtime')]},
 {'id': 'selector-minicpm-v1', 'title': '最新MiniCPM5-1B的相同请求对比', 'direction': '替换为实际1.08B模型与其官方分词器／模板，原规则／成对展示两组各101请求、候选、原十例、完整OCR、128token解码预算不变。', 'finding': '符合80／77，有害11／4、漏纠10／20；同请求Qwen对照92／94、有害3／4、漏纠6／3。必要修复37／27（各47）；两组均不采用。', 'scope': '202个完整响应全部review；55个变化含6改善、35退化、14个均允许的拼写规范差异。457个历史实际Go准备复用，914结果含712无候选跳过，无新增正文／ASR／Vision或语义拒绝回退。', 'runs': [('rewrite-selector-minicpm-v1/comparison-run', 'staged')]},
 {'id': 'minicpm-f16-preflight-v1', 'title': 'MiniCPM5-1B F16的原生与缓存探针', 'direction': '实际1.08B、官方F16导出；Metal99与CPU层0各六次。完整输入、原规则与schema／无schema探针均保持不变。', 'finding': '12次协议正常、缓存原始输出一致；8次正向纠错通过，4次现金含义仍被误改。原Q4的12次任务判断均失败，F16修复漏纠，但没有解决含义冲突。', 'scope': '12个完整响应全部review；两个源的重复诊断，不是12条独立语义样本。实际25/25与0/25GPU层、非思考前缀已核对，零新增正文／ASR／Vision。', 'runs': [('rewrite-selector-minicpm-f16-v1/runtime-view', 'runtime')]},
 {'id': 'selector-minicpm-f16-v1', 'title': 'MiniCPM5-1B F16与Q4的精度／导出对比', 'direction': '两组各101个完整请求与既有Q4逐项一致，保留原十例、全部OCR、候选、128token预算及原生参数。实际参数、张量名称／形状、完整分词器元数据和模板均相同。', 'finding': '两组均符合82/101，有害15／17、漏纠4／2，必需修复43／45（各47）；Q4符合80／77，有害11／4。纠错增多但错误覆盖原意也更多，两组仍不采用。', 'scope': '202个完整响应全部review；47个Q4差异含25改善、18退化、4个均允许的规范差异。457个实际Go准备复用，914结果含712无候选跳过；零新增正文／ASR／Vision，无语义过滤或原文回退。', 'runs': [('rewrite-selector-minicpm-f16-v1/run', 'staged')]},
 {'id': 'youtu-preflight-v1', 'title': 'Youtu1.96B的原生与缓存探针', 'direction': '官方Q8、实际1,961,560,064参数；Metal99与CPU层0各六次，原完整请求与schema／自由输出对照不变。', 'finding': '12次协议正常，实际33/33与0/33GPU层、模板和非思考前缀正确；缓存原始输出一致。4次现金含义保护正确，8次正向纠错漏改。', 'scope': '12个完整响应全部review；两个暴露源的重复诊断，兼容性只允许后续对比。自定义许可仅获本地研究授权，零新增正文／ASR／Vision。', 'runs': [('rewrite-selector-youtu-v1/runtime-view', 'runtime')]},
 {'id': 'selector-youtu-v1', 'title': 'Youtu1.96B的相同纠错请求比较', 'direction': '两组各101个完整请求与既有Qwen控制逐项一致，保留原十例、全部OCR、候选、128token预算和原生参数；比较新的非Qwen模型。', 'finding': '两组符合75／84（各101），有害3／4、漏纠23／13，必需修复24／34（各47）。旧控制符合92／94、有害3／4；58个差异含5改善、32退化，仍不采用。', 'scope': '202个完整响应全部review；29个不同失败组合和58个差异单独复核。复用457个实际Go准备、914结果含712无候选跳过。零新增正文／ASR／Vision，无语义过滤或原文回退。', 'runs': [('rewrite-selector-youtu-v1/run', 'staged')]},
 {'id': 'lfm25-preflight-v1', 'title': 'LFM2.5 1.17B的原生与缓存探针', 'direction': '官方Q8、实际1,170,340,608参数；Metal99与CPU层0各六次。保留原完整请求、schema和自由输出对照。', 'finding': '12次协议正常、缓存原始输出一致；实际17/17与0/17GPU层，嵌入模板和非思考前缀正确。8次正向纠错正确，4次明确现金含义被误改。', 'scope': '12响应全部review；两个暴露源的重复诊断，兼容性仅允许后续对比。元数据日志截断出不完整UTF8，保留原始字节并精确检查，模型响应JSON均正常。', 'runs': [('rewrite-selector-lfm25-v1/runtime-view', 'runtime')]},
 {'id': 'selector-lfm25-v1', 'title': 'LFM2.5 1.17B的相同纠错请求比较', 'direction': '两组各101个完整请求与既有Qwen控制逐项一致，原十例、完整OCR、候选、schema、128token预算和原生参数不变。', 'finding': '两组均符合79/101，有害22、漏纠0，必需纠错47/47。旧控制符合92／94、有害3／4；46个差异含9改善、37退化。全部修复不等于语义正确，两组仍不采用。', 'scope': '202响应、21个不同失败组合和46个差异全部review。复用457个Go准备、914结果含712无候选跳过。零新增正文／ASR／Vision，无语义过滤或原文回退。', 'runs': [('rewrite-selector-lfm25-v1/run', 'staged')]},
 {'id': 'selector-demo-order-v1', 'title': '只调整原十例顺序：保留原句示例放最后', 'direction': '四组同模型同预算；将原十个完整问答示例由0…9调整为0,1,2,3,7,8,9,4,5,6。规则、示例内容与数量、完整原文和OCR均不变，隔离既有示例顺序。', 'finding': '符合92／78／94／88（各101），有害3／2／4／2、漏纠6／21／3／11。27个变化含3改善、23退化、1允许的规范化变化。更保守却明显漏纠，两版均不采用。', 'scope': '404响应、25个不同失败组合及27个差异全部review；457次实际Go准备四组复用。808次本地模板／分词检查零生成，零新增正文／ASR／Vision。无语义过滤或原文回退。', 'runs': [('rewrite-selector-demo-order-v1/run', 'staged')]},
 {'id': 'd1-transcript-v1', 'typedGroup': 'transcript', 'title': 'd1 0.38B：聊天记录装入原生 state', 'direction': '保留原22条消息在 state.originalMessages，使用原生 choice 判断。', 'finding': '两种配置均3/6正确，三条应保留反例全部误改。问题未截断，聊天封装不适配。', 'scope': '12次真实原生判断（6源×Metal／CPU），生成token为0但实际模型响应为12；全部亲自review。不是聊天补全或最终正文；无新增ASR／Vision／Go清理／生产变更。', 'runs': []},
 {'id': 'd1-structured-v1', 'typedGroup': 'structured', 'title': 'd1 0.38B：原生结构化输入与局部替换选项', 'direction': '完整当前输入和十例解析为结构化 state，长系统规则移到问题中。', 'finding': '两种配置均2/6正确，既漏纠又误改；问题实际被96-token上限截断，不采用。', 'scope': '12次真实原生判断（6源×Metal／CPU），生成token为0但实际模型响应为12；全部亲自review。不是聊天补全或最终正文；无新增ASR／Vision／Go清理／生产变更。', 'runs': []},
 {'id': 'd1-sentence-v1', 'typedGroup': 'sentence', 'title': 'd1 0.38B：原生结构化输入与整句选项', 'direction': '沿用完整十例及当前上下文，把选项改为原句与单处替换后的完整句。', 'finding': '两种配置均3/6正确；仍被96-token问题上限截断，漏改loadCache并误改软件配置，不采用。', 'scope': '12次真实原生判断（6源×Metal／CPU），生成token为0但实际模型响应为12；全部亲自review。不是聊天补全或最终正文；无新增ASR／Vision／Go清理／生产变更。', 'runs': []},
 {'id': 'd1-compact-structured-v1', 'typedGroup': 'compact-structured', 'title': 'd1 0.38B：去掉示例的原生局部选项', 'direction': '移除十例和相应说明短语，保留完整当前原文、OCR及请求中的旧规则。', 'finding': '两种配置均2/6正确；仍有问题截断，有害2、漏纠2（各6），不采用。', 'scope': '12次真实原生判断（6源×Metal／CPU），生成token为0但实际模型响应为12；全部亲自review。不是聊天补全或最终正文；无新增ASR／Vision／Go清理／生产变更。', 'runs': []},
 {'id': 'd1-compact-sentence-v1', 'typedGroup': 'compact-sentence', 'title': 'd1 0.38B：去掉示例的原生整句选项', 'direction': '与整句选项比较，仅去掉示例及相应说明短语，当前原文和OCR完整保留。', 'finding': '两种配置均6/6正确，但长规则被截断；六个已暴露样本不足验收，不能按完整提示词成功采用。', 'scope': '12次真实原生判断（6源×Metal／CPU），生成token为0但实际模型响应为12；全部亲自review。不是聊天补全或最终正文；无新增ASR／Vision／Go清理／生产变更。', 'runs': []},
 {'id': 'd1-policy-v1', 'typedGroup': 'policy', 'title': 'd1 0.38B：完整规则放入 state.policy', 'direction': '按固定原生训练预算封装：完整规则进 state.policy，短问题配合完整原句／替换句选项。', 'finding': '实际无截断，两种配置均3/6正确，有害2、漏纠1（各6）。修复截断不等于质量过关，本版不扩大跑测。', 'scope': '12次真实原生判断（6源×Metal／CPU），生成token为0但实际模型响应为12；全部亲自review。不是聊天补全或最终正文；无新增ASR／Vision／Go清理／生产变更。', 'runs': []},
]
GATES=[
 ['自然结构与场景适配','部分改善','最新诊断401源链路：构造281条有272条可用，自然120条有75条可用；旧24对照可用21→20，语义32对照仍16条可用，仍有漏纠错。'],
 ['原意、否定、条件与请求完整保留','部分改善','最新历史401正文400确认、1源文待核验；语义32对照保真29→22、误改3→10，旧24对照另新增2条误改，不能上线。'],
 ['OCR 正向纠错与正确文本保护','部分改善','新增LFM2.5实际1.17B两组均符合79/101，虽47/47必需修复完成，但各有22条有害修改。错误OCR仍会覆盖明确原意，无新增正文，不能替代完整链路验收；跨文字母术语仍待覆盖。'],
 ['提示词封装','部分改善','多轮封装结果并不一致；分离参考回合已退化，移除唯一标签仍未消除误改。仍需优化忠实度、纠错与结构。'],
 ['真实 ASR／OCR 配对证据','部分补齐','已对120条历史自然ASR源执行13组共1560次候选；未新增ASR或真实窗口OCR，仍需实际语音／上下文链路。'],
 ['原语言与中英混输保留','待验收','中英文自然源已实测，最新语系封装在120条保留语言；完整中英混输、独立语义与更多场景仍待验证。'],
 ['其他生产场景与逐字模式','待验收','当前研究只验证chat/email/work/prompt；search/list与verbatim保护仍需候选实测。'],
 ['计时一致性','待校准','历史396次原生观察中9对引擎阶段时间超过外层请求时间；保留原始值，尚无历史睡眠归因或延迟SLO证明。'],
 ['独立自然数据质量审查','待补齐','助手逐条 review 已记录；不冒充独立盲评。'],
 ['macOS 原生生命周期与离线','待验收','候选需验证取消、焦点／版本变化、一次提交与离线链路。'],
 ['端到端延迟与资源','待验收','最新诊断未预热构造281文本链路p95约2018ms、自然120约1573ms；旧小集预热观察单列，不替代当前ASR并发、原生写入或16GB验收。'],
 ['生产集成与回归','未完成','实验候选尚未替换生产模板与校验策略。'],
]

METRICS={
 'd1-transcript-v1':'固定官方Q8，文本组件380,732,161参数、全多模态家族587,161,089；实际19/19与0/19GPU层。六源选择一致但概率不同，不能声称数值或校准对齐。原生问题96token／两选项描述各45token独立于总上下文；36份分段分词证据，60次旧用量和50段旧原生提示完全匹配；新policy十二次用量亦匹配，无截断。机械应用选择的文本仅为审阅示意，未执行排版或交付。',
 'd1-structured-v1':'固定官方Q8，文本组件380,732,161参数、全多模态家族587,161,089；实际19/19与0/19GPU层。六源选择一致但概率不同，不能声称数值或校准对齐。原生问题96token／两选项描述各45token独立于总上下文；36份分段分词证据，60次旧用量和50段旧原生提示完全匹配；新policy十二次用量亦匹配，无截断。机械应用选择的文本仅为审阅示意，未执行排版或交付。',
 'd1-sentence-v1':'固定官方Q8，文本组件380,732,161参数、全多模态家族587,161,089；实际19/19与0/19GPU层。六源选择一致但概率不同，不能声称数值或校准对齐。原生问题96token／两选项描述各45token独立于总上下文；36份分段分词证据，60次旧用量和50段旧原生提示完全匹配；新policy十二次用量亦匹配，无截断。机械应用选择的文本仅为审阅示意，未执行排版或交付。',
 'd1-compact-structured-v1':'固定官方Q8，文本组件380,732,161参数、全多模态家族587,161,089；实际19/19与0/19GPU层。六源选择一致但概率不同，不能声称数值或校准对齐。原生问题96token／两选项描述各45token独立于总上下文；36份分段分词证据，60次旧用量和50段旧原生提示完全匹配；新policy十二次用量亦匹配，无截断。机械应用选择的文本仅为审阅示意，未执行排版或交付。',
 'd1-compact-sentence-v1':'固定官方Q8，文本组件380,732,161参数、全多模态家族587,161,089；实际19/19与0/19GPU层。六源选择一致但概率不同，不能声称数值或校准对齐。原生问题96token／两选项描述各45token独立于总上下文；36份分段分词证据，60次旧用量和50段旧原生提示完全匹配；新policy十二次用量亦匹配，无截断。机械应用选择的文本仅为审阅示意，未执行排版或交付。',
 'd1-policy-v1':'固定官方Q8，文本组件380,732,161参数、全多模态家族587,161,089；实际19/19与0/19GPU层。六源选择一致但概率不同，不能声称数值或校准对齐。原生问题96token／两选项描述各45token独立于总上下文；36份分段分词证据，60次旧用量和50段旧原生提示完全匹配；新policy十二次用量亦匹配，无截断。机械应用选择的文本仅为审阅示意，未执行排版或交付。',
 'selector-demo-order-v1':'202个当前控制复现旧原始选择与编辑文本；202个重排请求逆置换后完全还原，22条消息与十例内容不变。最长完整输入1891＋128≤4096，404次实际token均与原生检查一致。每组12个歧义schema强制keep不代表自主判断。原冻结损坏数据测试存在不存在的组名，失败保留、v2修正测试目标，未改推理或评分。当前无16GB实机，按用户意见继续32GB开发机验证，最低配置验收仍未执行。',
 'lfm25-preflight-v1':'原生实际模板与官方GGUF嵌入1783字符一致，上游5487字符模板不同。显式移除不输出文本的追踪标记后，本地Jinja202个文本请求渲染一致；新增生成0次，不能替代原生分词／官方Transformers／数值对齐。',
 'selector-lfm25-v1':'每组89个可自主选择的唯一候选均替换，12个歧义schema强制原句。质保书被改成指保书、地点被改成行为；正确修复覆盖率高但语义边界失败。自定义许可生产采用前另核对，实际16GB／独立评审／完整交付未验收。',
 'youtu-preflight-v1':'所有探针均输出keep：明确现金含义正确，但上下文支持的load cash→loadCache未纠正。自由输出与schema一致，不能归因于输出schema强制；不作为12个独立质量样本。',
 'selector-youtu-v1':'保守输出不能等同于正确：会议记要、验受报告等充分语境仍漏纠；错误OCR仍会把服务器部署写成培植清单。每组12个原句唯一选项不算自主判断。自定义许可包含地域限制，生产采用前另行核对；实际16GB及完整交付未验收。',
 'minicpm-f16-preflight-v1':'正向load cash→loadCache由漏纠变为正确；明确现金／非缓存的read cash仍错误替换为readCache，去除schema也失败。兼容性通过不能作为语义或上线通过。',
 'selector-minicpm-f16-v1':'202个完整请求、准备结果及输入token计数与Q4一致。提高精度并非单调改善；普通无关OCR也会把大理改成搭理。每组12个原句唯一选项的歧义病例不算自主判断；尚无数值转换血缘证明，不能把全部差异单独归因为量化。',
 'minicpm-preflight-v1':'实际Metal25/25与CPU0/25；嵌入模板与固定官方文件一致，运行时解析器仅移除末尾一个LF。原生非思考前缀已验证，去除schema仍漏纠／误改；12次不是12条独立语义样本。',
 'selector-minicpm-v1':'两组必需修复37/47、27/47；原有12个原句唯一选项的歧义病例每组保留，不计自主判断。202个请求与既有Qwen逐项完全一致；模型／架构／分词器／模板一起变化，不能单独归因于参数量。',
 'selector-contrast-demo-v1':'202个当前对照的请求／准备与旧实际相同，原始输出复现202/202。202个新请求去除8条附加示例消息后完全还原；6个变化逐项review。最长完整输入2482＋输出预算128≤4096；全部实际推理token数与原生预检一致。每组12个原句唯一选项的歧义病例不代表自主语义判断。',
 'selector-unique-label-v1':'202个当前对照与旧实际请求／准备结果相同，原始输出复现202/202。202个新请求补回unique后完全还原对照，4个变化逐项review。保留每组12个原句唯一选项的歧义病例，不把协议限制算作自主语义判断。零新增最终正文。',
 'selector-reference-turn-v1':'202个当前对照的请求／候选与旧实际请求相同，原始输出复现202/202。202个分离请求还原context后与并行对照完全相同；16个选择变化逐项review。每组12个原句唯一选项的歧义病例不算自主语义判断，零新增正文。 本次串行请求中位245／1102／496／1230ms；分离回合的预填充工作更多，不代表端到端或16GB延迟。',
 'instruction-language-qwen-v1':'唯一改变为messages[0].content；十个中文示例、助手答案、当前源和所有其他请求字段完整保留。9条原规则英译均有逐条对齐与来源SHA。必需纠正44／45（各47），仍有10／12有害，均不通过。',
 'instruction-language-hybrid-v1':'同模型、同聊天模板／候选／解码／Metal设置，只比较既有中文规则和本轮英文翻译。两组必需修复均47/47，但有害从20／19增至21／22；含义仍会被错误OCR覆盖，不进入正文或上线验收。',
 'hybrid-preflight-v1':'实际Metal41/41和CPU层0/41均匹配官方模板。去除schema后现金定义错误仍出现，不能归为语法约束强迫。12次探针与202次选择分开计数，无完整交付／16GB验收。',
 'selector-hybrid-v1':'202个请求／准备结果与已有Qwen逐项完全相同，无重跑对照。两组漏纠均0，仍有20／19个有害选择。12个原句唯一选项的歧义病例每组沿用，不代表模型自主判断；无语义输出拒绝或原文回退。',
 'selector-sentence-choice-v1':'202个当前对照与旧请求／候选完全相同，原始选择复现202/202。101组A/B输入只交换选项标签及相应示例／schema；归一操作一致77/101，其中20组通过状态不同、4组双方符合但规范化选择不同。完整OCR、十个示例源及候选不变，无语义输出拦截或原文回退；原始A/B与归一keep/replace分别保留。',
 'selector-source-last-v1':'202个当前对照的完整请求／候选与旧实际请求相同，原始选择和改后源文复现202/202。202个字段顺序对照恢复source原位置后请求完全一致；17个选择变化逐项review。原文末尾必需纠正44/47、成对加末尾46/47，但有害替换5／12例，均不通过。',
 'selector-referent-v1':'202个本轮对照与旧128token请求仅上限不同，原始选择／改后源文逐字一致202/202；新说明都按referent→choice顺序生成，但这不证明内部推理。说明整例复核78／80（各101），与选择联合通过78／80；每组仍有必要纠错遗漏与错误OCR替换。',
 'selector-model-cpu-v1':'相同请求的既有Qwen原规则／成对展示为92／94符合、3／4有害选择；Granite为84／86符合、16／13有害选择。必需候选项纠正46／45（各47），更多纠错不能掩盖语义误改。两组均会无依据把现金含义的read cash改成readCache。',
 'granite-runtime-v1':'实际1,631,750,144参数、官方Q4导出；7个响应中3个CPU诊断任务匹配、4个Metal输出退化。另6个HTTP错误（首次1＋诊断5）完整保留。未执行的202次不能计作完成。',
 'selector-screen-v1':'原规则与单向预览各101请求／决定／编辑原文完全匹配对应旧完整链路，共202个对照。成对展示必需纠正44/47候选项，仍不通过；专名规则46/47却有17有害选择。22个同字母规范化源允许keep或replace，不回写历史判分。',
 'selector-preview-v1':'14个选择全部由keep变replace：人工复核5个正确、9个有害。旧24对照保真24→22／可用21→20，语义32保真29→22／可用仍16；历史可用346→347。总可用仍383/457，不能掩盖误改3→12。',
 'selector-semantic-v2':'当前101选择有92符合要求；旧24对照保真24、可用21，纠错7/8。新增32对照保真29、可用16，较上一版多1误改；完整历史400确认／1源文待核验／346可用。不得以正例上涨掩盖错误OCR仍可影响选择。',
 'selector-semantic-v1':'新对照保真26→30/32、可用14→16/32；旧24可用16→15，误改消失但漏纠增加。完整历史400确认／1待核验／338可用。无语义过滤或原文回退；完整609输出、场景、实际请求及模板身份异常均展示。',
 'tone-chain-v1':'历史仅新增1个候选，验受→验收仍被keep；旧64次选择请求／决定完全一致。人工纠错8例只多修复质保书1例，新增1例跨语义误改。历史构造281文本链路p95约2652ms，自然120约1081ms；未预热，不与旧小集SLO直接比较。',
 'work-scope-v3':'新背景示例修复任务列表遗漏，但新增负责人不变遗漏；历史总分持平。口头重复与专属日期分离仍存在，全部126原始输出保留。',
 'work-scope-v4':'历史保真93→94/94、可用85→87/94；同16对照16/16。小陈／小刘等4例恢复负责人／日期归组，另3例日期仍分离。余7条可用性不足来自未解术语，不计为新模型幻觉。',
 'history-format-v2':'构造281：280确认、1明确失败、0待核验、267可用；自然120：119确认、1待核验、75可用。英文日期改口改善，中文仍保留改口过程。新任务列表漏接口恢复／回填状态，不以总分上涨掩盖退化。',
 'history-chain-v1':'构造281：278确认、2失败、1待核验、260可用；自然120：119确认、1待核验、75可用。共同时间作用域仍可能缩窄；显式not20遗漏但最终2秒正确。取消请求语气保留待核验；完整旧判断和复核修订可查。',
 'metal-primed-v1':'两顺序正式348次及预热32次全部review；正式每组174次缓存均命中、原始输出匹配基线174/174。每组36场景保真／可用、纠错11/11；自然119确认、1待核验、75可用。场景p95约836／1470ms，预热就绪成本单列。',
 'metal-prefix-v1':'174/174请求去除缓存开关后匹配，174/174原始输出一致；164次实际缓存命中、90542个复用token。36场景保真／可用、纠错11/11；自然119确认、1待核验、75可用。首次未命中均保留，场景p95约2.42秒。',
 'metal-chain-v1':'18/18实际选择符合要求；36/36场景正文保真／可用，必要纠错11/11；自然119确认、1待核验、75可用，语言120/120。清理IPC至正文场景p50约1.87秒、p95约2.92秒；不含ASR和原生写入。',
 'metal-regression-v1':'单排版请求p50约1039ms、p95约1463ms、p99约1603ms；4个实际进程均记录25/25层GPU放置。35/36冻结口径忠实／可用。',
 'natural-format-v1':'120条：109已确认忠实、8明确失败、3待核验；72可用。语言保留116/120，4次英译中。37条输出仍受残缺／语法源文限制，非所有输入困难的排他计数。',
 'language-profile-v1':'场景36/36忠实／可用、纠错11/11；自然117忠实、2失败、1待核验、73可用，语言118/120。局部通过不掩盖新反向翻译。',
 'language-profile-v2':'场景36/36忠实、35/36可用、纠错11/11；自然119已确认忠实、0明确失败、1待核验、75可用，语言120/120。未把复制残缺源文算作自然成功。',
 'cleanup-copula-v1':'156条真实Go清理、2条文本变化（比v1新增1条），UTF-8删除范围回放一致。新增10条正反例；核心测试和5秒模糊测试通过。',
 'metal-preflight-v1':'1条完整输出已review，负责人／任务／日期正确；同请求输入586token、输出22token、未用缓存。实际措辞有变化，必须另测完整质量。',
 'formatter-v3':'按冻结强调口径忠实34→35/36、场景可用33→35/36；必要拼写纠错均11/11。新产生任务范围变化已消失；唯一未过项不是数值改错，而是省略冗余排除强调。',
 'bounded-edit-v2':'中间36/36选择符合要求；必要拼写纠错11/11保留至正文。最终34/36忠实、场景可用33/36；显式对比与任务范围要求仍有损失。',
 'bounded-edit-v1':'中间选择33/36符合要求、必要纠错8/11；最终正文忠实28/36、场景可用27/36、必要纠错6/11。JSON协议有效不代表正文正确，两阶段不能计为72条成功改写。',
 '2b-precision-v1':'Q4→Q8：忠实34→32/36，场景可用34→32/36，必要纠错11→9/11；36条请求／编码及输入token计数一致。Q8最高采样workerRSS约4.42GB，非峰值或整机内存；无独立转换／数值权重血缘证明，不作严格量化因果结论。',
 'data-object-v1':'复用基线／小写／JSON：忠实34／34／31（各36），场景可用34／34／31，必要纠错11／11／9（各11）。小写版8条无上下文请求及输出逐字一致；其定义反转不能被相同总分掩盖。',
 'private-spans-v1':'按冻结的保守拼写口径：忠实31→34/36，场景可用30→34/36，必要纠错10→11/11；无截断或输出拒绝。新增OCR声明是确定退化；cash/cache定义已保留，但readCache不满足用户允许的同字母规范化，不能称作定义反转。',
 'current-task-v1':'保护／尾部任务／取消拒绝：忠实32／26／25（各36），场景可用31／23／22，必要纠错均9/11；协议失败2／1／0，原始正文正确却被协议拒绝均0。',
 'system-task-probe-v1':'同12条难例：原保护8/12忠实、7/12可用；尾部任务4/12、3/12；取消拒绝3/12、2/12；system版8/12、7/12，必要纠错1/2。无截断／输出拒绝。',
 'local-protection-stress-v1':'原冻结口径完整正文忠实1→4/8、场景可用0→3/8；必要纠错交付2→0/2；保护组2次编码协议失败，全部原始输出保留。有效协议不等于改写正确。',
 'local-protection-v1':'Qwen直接→局部保护：忠实25→28/28，场景可用25→28/28，纠错均9/9；25条无编码对照请求与正文逐字一致。Granite忠实19/28、场景可用18/28、纠错7/9。聊天仍有轻微语气词残留。',
 'source-plan-v1':'直接正文→编辑方案：忠实24→19/28，场景合适20→6/28，必要纠错8→0/9；协议失败与截断均0。CPU请求p95约6992→13924ms，串行顺序有漂移，非语法单因素对照。',
 '2b-contrast-v1':'忠实均15/16，场景合适均13/16，必要纠错均3/3；歧义仍失败。',
 'pronunciation-evidence-v1':'忠实20→21/24，场景合适18→19/24，必要纠错6→7/7；三条歧义仍误选。8条相同请求负对照也变慢，不能把时间差归因于证据注入。',
 'small-prompt-v1':'忠实均14/16，场景合适10→11/16；getProfile纠错出现倒退，未通过预定选择门槛。',
 'small-cache-v1':'正式请求p95约2644→701ms；缓存命中551／572token；原始文本一致14/16，顺序一致16/16。',
 'small-cache-failed-v1':'原生启动失败、完成生成0；未执行的40次计划不计入样本数。',
 'small-cache-parity-v2':'正式请求缓存开关逐字一致16/16；缓存p95约1614ms（正序921、逆序1614ms），预热另列。',
 'small-models-v1':'0.8B／2B：忠实22／23（各24），场景合适20／23，纠错均4/4；无缓存CPU请求p95约2029／4466ms，最高采样RSS约1.65／3.16GB。2B的冗余单条列表另记为轻微风格问题。',
 'model-screen-v1':'忠实 6→6/8，纠错 1→2/2；请求 p50 696→1580 ms、p95 872→2072 ms。',
 '4b-packaging-v1':'忠实：分区 6/8、JSON 7/8、XML 5/8；纠错分别 2/2、1/2、1/2；未扩大验收。',
 '4b-entity-v1':'忠实 7→6/8；纠错仍为 1/2；原文被 OCR 替换的失败重新出现。',
 'alignment-v1':'事后复核原始新集：忠实 24→12/24、自然场景组织 14→12/24；与原代理分分别呈现，不追改原指标。',
 'entity-evidence-v1':'两版开发均忠实 21/26，纠错 3/5，正确原文控制 5/8；Vision 两级的已知答案修复均 4/5，含示例记忆，不能据此声称 OCR 收益。',
 'task-wrapper-v1':'旧集忠实 22→21/24；新集忠实 18→18/24、场景合适 14→16/24；请求 p95 1150→1590 ms。',
 'isolation-v1':'开发忠实：基线 16/18，英文／中文均 14/18；场景合适 12→10/18；纠错 3→2/5。',
 'scene-v2':'新集 32 条：忠实 28→26；忠实且场景合适 9→22。',
 'structure-v3':'非示例 6 条开发探针：忠实 5→6；场景合适 2→6。',
 'context-v1':'必须纠错 4 条：1→2；负向控制 8 条：6→7。',
 'ungated-v1':'候选新集 24 条：strict／none 忠实交付 22／21，场景合适 20／19。',
 'ocr-v2':'必须纠错：旧集 0→2/4；新集 2→4/8。新集忠实原始输出 12→16/24。',
 'packaging-v1':'XML→分区：旧集忠实 17→22/24；新集忠实 17→20/24，纠错 3→4/7，场景合适 13→16/24。',
}

def read(path):return json.loads(path.read_text())
def digest(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()

def load_report(relative,alias,filename='report.py'):
    path=ROOT/'benchmarks/reproduction'/relative/filename
    previous_path=list(sys.path);previous_run=sys.modules.pop('run',None)
    sys.path.insert(0,str(path.parent))
    try:
        spec=importlib.util.spec_from_file_location(alias,path)
        module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        return module
    finally:
        sys.path[:]=previous_path
        sys.modules.pop('run',None)
        if previous_run is not None:sys.modules['run']=previous_run

def collect():
    entity=load_report('rewrite-entity-evidence-v1','entity_evidence_report')
    entity.checked(ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/dev','dev')
    entity.checked(ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/vision','ocr')
    retrospective=load_report('rewrite-retrospective-v1','retrospective_report')
    retrospective.checked('dev');retrospective.checked('test')
    for name,folder in [('rewrite-model-screen-v1','development'),('rewrite-4b-packaging-v1','dev'),('rewrite-4b-entity-v1','dev')]:
        load_report(name,name.replace('-','_')).checked(ROOT/'benchmarks/runs'/name/folder)
    small=load_report('rewrite-small-models-v1','small_model_report')
    for model in small.MODELS:small.checked(ROOT/'benchmarks/runs/rewrite-small-models-v1'/model)
    for name,folder in [('rewrite-small-prompt-v1','dev'),('rewrite-small-cache-v1','run'),('rewrite-small-cache-parity-v2','run'),('rewrite-2b-contrast-v1','dev'),('rewrite-pronunciation-evidence-v1','dev'),('rewrite-source-plan-v1','dev')]:
        load_report(name,name.replace('-','_')).checked(ROOT/'benchmarks/runs'/name/folder)
    protection=load_report('rewrite-local-protection-v1','local_protection_report')
    for arm in protection.runner.ARMS:protection.checked(ROOT/'benchmarks/runs/rewrite-local-protection-v1'/arm)
    stress=load_report('rewrite-local-protection-stress-v1','local_protection_stress_report')
    for arm in stress.ARMS:stress.checked(ROOT/'benchmarks/runs/rewrite-local-protection-stress-v1'/arm)
    current=load_report('rewrite-current-task-v1','current_task_report')
    for arm in current.ARMS:current.checked(ROOT/'benchmarks/runs/rewrite-current-task-v1'/arm)
    load_report('rewrite-system-task-probe-v1','system_probe_report').checked(ROOT/'benchmarks/runs/rewrite-system-task-probe-v1/dev')
    load_report('rewrite-cleanup-preflight-v1','cleanup_preflight','verify.py').checked()
    private=load_report('rewrite-private-spans-v1','private_spans_report')
    for arm in private.ARMS:private.checked(ROOT/'benchmarks/runs/rewrite-private-spans-v1'/arm)
    objects=load_report('rewrite-data-object-v1','data_object_report')
    for arm in objects.ARMS:objects.checked(ROOT/'benchmarks/runs/rewrite-data-object-v1'/arm)
    objects.baseline()
    precision=load_report('rewrite-2b-precision-v1','precision_report')
    for arm in precision.ARMS:precision.checked(ROOT/'benchmarks/runs/rewrite-2b-precision-v1'/arm)
    precision.baseline()
    load_report('rewrite-bounded-edit-v1','bounded_report').checked()
    load_report('rewrite-bounded-edit-v2','bounded_v2_report').checked()
    load_report('rewrite-formatter-v3','formatter_v3_report').checked()
    load_report('rewrite-metal-preflight-v1','metal_probe').checked()
    contrast_demo_summary=load_report('rewrite-selector-contrast-demo-v1','selector_contrast_demo').collect()[0]
    unique_label_summary=load_report('rewrite-selector-unique-label-v1','selector_unique_label').collect()[0]
    reference_turn_summary=load_report('rewrite-selector-reference-turn-v1','selector_reference_turn').collect()[0]
    instruction_language_summary=load_report('rewrite-selector-instruction-language-v1','selector_instruction_language').collect()[0]
    d1_summary,d1_items=load_report('rewrite-d1-evidence-v1','native_d1_evidence').collect()
    demo_order_summary=load_report('rewrite-selector-demo-order-v1','selector_demo_order').collect()[0]
    lfm25_summary=load_report('rewrite-selector-lfm25-v1','selector_lfm25').collect()[0]
    youtu_summary=load_report('rewrite-selector-youtu-v1','selector_youtu').collect()[0]
    minicpm_f16_summary=load_report('rewrite-selector-minicpm-f16-v1','selector_minicpm_f16').collect()[0]
    minicpm_tokenizer=load_report('rewrite-minicpm-tokenizer-v1','minicpm_tokenizer').checked()
    minicpm_summary=load_report('rewrite-selector-minicpm-v1','selector_minicpm').collect()[0]
    hybrid_summary=load_report('rewrite-selector-hybrid-v1','selector_hybrid').collect()[0]
    sentence_choice_summary=load_report('rewrite-selector-sentence-choice-v1','selector_sentence_choice').collect()[0]
    source_last_summary=load_report('rewrite-selector-source-last-v1','selector_source_last').collect()[0]
    referent_summary=load_report('rewrite-selector-referent-v1','selector_referent').collect()[0]
    cpu_model_summary=load_report('rewrite-selector-model-cpu-v1','granite_cpu').collect()[0]
    runtime_diagnostics=load_report('rewrite-selector-model-v1','granite_runtime').checked()[0]
    screen_summary=load_report('rewrite-selector-screen-v1','selector_screen').collect()[0]
    preview_summary=load_report('rewrite-selector-preview-v1','selector_preview').collect()[0]
    semantic_v2_summary=load_report('rewrite-selector-semantic-v2','selector_semantic_v2').collect()[0]
    semantic_summary=load_report('rewrite-selector-semantic-v1','selector_semantic').collect()[0]
    tone_summary=load_report('rewrite-tone-chain-v1','tone_chain').collect()[0]
    work_v3_summary=load_report('rewrite-work-scope-v3','work_scope_v3').collect()[0]
    work_v4_summary=load_report('rewrite-work-scope-v4','work_scope_v4').collect()[0]
    history_format_summary=load_report('rewrite-history-format-v2','history_format_v2').collect()[0]
    history_summary=load_report('rewrite-history-chain-v1','history_chain').collect()[0]
    primed_summary=load_report('rewrite-metal-primed-v1','metal_primed').collect()[0]
    load_report('rewrite-metal-prefix-v1','metal_prefix').checked()
    load_report('rewrite-metal-chain-v1','metal_chain').checked()
    load_report('rewrite-metal-regression-v1','metal_regression').checked()
    load_report('rewrite-natural-format-v1','natural_format').checked()
    load_report('rewrite-language-profile-v1','language_profile_1').checked()
    load_report('rewrite-language-profile-v2','language_profile_2').checked()
    copula=load_report('rewrite-cleanup-copula-v1','copula_cleanup','verify.py').checked()
    natural_text=load_report('rewrite-natural-text-review-v1','natural_text_review','verify.py').checked()
    load_report('rewrite-small-cache-parity-v1','cache_failed_report').checked()
    natural=load_report('rewrite-natural-source-audit-v1','natural_source_audit','verify.py').checked()
    timing_audit=load_report('rewrite-timing-audit-v1','native_timing_audit','verify.py').checked()
    rounds=[];cases=[];sources=[]
    for definition in ROUND_DEFINITIONS:
        entry={k:v for k,v in definition.items() if k!='runs'};entry['count']=0;entry['metrics']=METRICS[definition['id']]
        if definition.get('typedGroup'):
            for native in [item for item in d1_items if item['group']==definition['typedGroup']]:
                row=native['row'];source=native['source'];audit=native['assessment'];budget=native['budget'];review=native['review']
                cases.append(dict(round=definition['id'],run='native-d1/'+native['group'],variant='metal' if row['gpuLayersRequested']==99 else 'cpu',id=row['id'],role='measured',cohort='six-source-native-decision-development',stage='typed-decision',scene=row['scene'],effectiveScene='native-spelling-decision',sceneEvidence='完整原生请求中记录的场景；模型未生成正文',assignment='测试用例指定，未测自动命中',template='tokenizer.chat_template.systemone/'+native['group'],wrapper='native-state/'+native['group'],model='d1-omni-600M-Q8_0 text component',source=source['source'],context=source['context'],output=json.dumps(row['response'],ensure_ascii=False,indent=2),cleanup=None,faithful=not audit['harmfulChoice'],recordedFaithful=not audit['harmfulChoice'],useful=None,selectionConforms=audit['choiceConforms'],selectionReviewBasis='explicit native choiceConforms; original typed answers and probabilities',issues=['unsupported-substitution'] if audit['harmfulChoice'] else ['missed-required-repair'] if audit['missedRepair'] else [],note=review['note'],strict='未启用',envelope='未启用',delivery=None,milliseconds=row['requestMs'],actualMessages=None,modelSource=source['prepared']['source'],speechCleanup=None,reviewQualifier=None,reviewCaveat=None,actualContextUsed=source['context'],spellingHypotheses=source['prepared'],actualNativeRequest=row['request'],effectiveNativePrompt=budget['effectivePrompt'],auditDerivedSpellingText=audit['derivedSpellingText'],nativeBudget=dict(inputTokens=budget['inputTokens'],outputTokens=0,questionLimit=96,optionLimit=45,truncatedPieces=budget['truncatedPieces'],modelGenerationsInBudgetCheck=0),runtimeConfiguration=dict(nativeCommand=native['actualRuntimeCommand'],rawResponsePath=native['rawResponsePath'],rowSha256=native['rowSha256'],newGoPreparations=0,formatterExecuted=False),performance=dict(inputTokens=row['response']['usage']['input_tokens'],outputTokens=0),sourcePreparation=dict(reference=row['sourceReference'],newGoExecuted=False,formatterExecuted=False)))
                entry['count']+=1
            rounds.append(entry)
            continue
        for relative,mode in definition['runs']:
            directory=ROOT/'benchmarks/runs'/relative
            paths=[directory/'results.jsonl',directory/'review.json',directory/'manifest.json']
            manifest=read(paths[2])
            if manifest['status']!='complete' or manifest['resultsSha256']!=digest(paths[0]):raise ValueError('Incomplete or changed run: '+relative)
            review=read(paths[1])
            if review['resultsSha256']!=digest(paths[0]):raise ValueError('Changed review source')
            keyfield='arm' if mode=='parity' else 'variant'
            judgments={(x[keyfield],x['id']):dict(x) for x in review['reviews']}
            clarification=directory/'review-clarification.json'
            if clarification.exists():
                amendment=read(clarification)
                if amendment['reviewSha256']!=digest(paths[1]):raise ValueError('Changed clarified review')
                correction=amendment['correction'];item=judgments[(correction['variant'],correction['id'])]
                item['issues']=[issue for issue in item['issues'] if issue!=correction['removeIssue']]
                sources.append(dict(path=str(clarification.relative_to(ROOT)),sha256=digest(clarification)))
            caveat_path=directory/'review-caveats.json'
            if caveat_path.exists():
                caveats=read(caveat_path)
                if caveats['reviewSha256']!=digest(paths[1]):raise ValueError('Changed review caveat source')
                for caveat in caveats['caveats']:
                    item=judgments[(caveat['variant'],caveat['id'])]
                    if caveat['rowSha256']!=item['rowSha256']:raise ValueError('Changed caveat row identity')
                    item['reviewCaveat']=caveat['note']
                sources.append(dict(path=str(caveat_path.relative_to(ROOT)),sha256=digest(caveat_path)))
            rows=[json.loads(line) for line in paths[0].read_text().splitlines()]
            if definition['id']=='metal-primed-v1':
                warm_paths=[directory/'warmups.jsonl',directory/'warmup-review.json',directory/'study.json']
                warm_review=read(warm_paths[1])
                if warm_review['resultsSha256']!=digest(warm_paths[0]):raise ValueError('Changed reviewed priming bytes')
                judgments.update({(j['variant'],j['id']):j for j in warm_review['reviews']})
                rows.extend(json.loads(line) for line in warm_paths[0].read_text().splitlines());paths.extend(warm_paths)
            if len(judgments)!=len(rows):raise ValueError('Incomplete review coverage')
            for row in rows:
                key=(row[keyfield],row['id']);item=judgments[key]
                if item['rowSha256']!=rowhash(row):raise ValueError('Reviewed output changed')
                if mode=='runtime':
                    cases.append(dict(reviewQualifier=None,reviewCaveat=None,round=definition['id'],run=relative,id=row['id'],variant=row['variant'],cohort=row['cohort'],model=row['modelId'],scene=row['scene'],effectiveScene=row['effectiveModelScene'],sceneEvidence='记录于实际诊断请求；未测试自动场景',assignment='运行故障诊断，未测自动命中',template=row['templateVersion'],source=row['source'],context=row['context'],output=row['response']['choices'][0]['message']['content'],faithful=None,useful=None,issues=[],note=item['note'],strict='未启用',envelope='未启用',milliseconds=row['generationMs'],stage='runtime-probe',probeConforms=item['probeConforms'],delivery=None,actualMessages=row['request']['messages'],modelSource=None,runtimeProbe={k:row[k] for k in ['rawReference','expectedDiagnosticChoice','actualNativeCommand','residentRssBytes']}))
                    entry['count']+=1
                    continue
                raw_output=(row['renderedBody'] if row['renderedBody'] is not None else ('[编码协议失败，无交付正文]' if mode=='protected' else '[方案构造失败，无交付正文]')) if mode in ['sourceplan','protected'] else row['response']['choices'][0]['message']['content'] if mode in ['gguf','staged'] else row['response'].get('text','')
                if mode=='staged' and row['stage']=='format':raw_output=row['renderedBody']
                gguf_template=None
                if mode=='gguf':
                    template_name=f"{row['variant']}-{row['scene']}.json" if definition['id'] in ['small-prompt-v1','2b-contrast-v1','pronunciation-evidence-v1'] else f"{row['scene']}.json"
                    template_study='rewrite-'+definition['id'] if definition['id'] in ['small-prompt-v1','2b-contrast-v1','pronunciation-evidence-v1'] else 'rewrite-small-models-v1'
                    gguf_template=ROOT/'benchmarks/reproduction'/template_study/'templates'/template_name
                    if digest(gguf_template)!=row['templateSha256']:raise ValueError('Wrong displayed GGUF template')
                faithful=item['proposalFaithful'] if 'proposalFaithful' in item else item['faithful']
                useful=item['proposalSceneUseful'] if 'proposalSceneUseful' in item else item.get('sceneUseful')
                scene='work' if mode=='parity' else row['scene']
                effective=row.get('effectiveModelScene')
                evidence='记录于原始推理行'
                if effective is None:
                    if mode not in ['direct','parity']:raise ValueError('Missing model scene')
                    effective=scene;evidence='由冻结 runner 的直接场景参数复原'
                accepted=row.get('strict')=='accepted' and row.get('envelope','accepted')=='accepted'
                delivery=None
                if 'cleanupFaithful' in item:
                    delivery=dict(text=row['response'].get('text','') if accepted else row['response'].get('cleanup',{}).get('text'),origin='模型改写' if accepted else '已完成的口述清理',faithful=faithful if accepted else item['cleanupFaithful'],note=item['note'] if accepted else item['cleanupNote'])
                if mode in ['gguf','sourceplan','protected']:delivery=dict(text=raw_output,origin='字面还原正文（不做输出拒绝）' if mode=='protected' and row['encoding'].get('deliveryPolicy')=='unfiltered' else '保护词还原正文（无语义校验，仅检查编码协议）' if mode=='protected' and row['encoding']['protectedTerms'] else '方案构造正文（无语义校验，仅检查JSON协议）' if mode=='sourceplan' and row['variant']=='plan' else '模型原始输出（未启用校验）',faithful=faithful,note=item['note'],policy='none')
                if mode in ['sourceplan','protected'] and row['renderedBody'] is None:delivery=None
                provenance=manifest.get('provenance',{});description=provenance.get('model',{})
                model=description.get('id') if isinstance(description,dict) else None
                if 'modelFiles' in row and row[keyfield] in provenance.get('modelFiles',{}):model=row[keyfield]
                performance={k:row['response'].get(k) for k in ['prefillMs','decodeMs','inputTokens','outputTokens']}
                if mode in ['gguf','sourceplan','protected','staged']:
                    model=manifest['model']['id'];timing=row['response']['timings'];performance=dict(cachedTokens=timing['cache_n'],prefillMs=timing['prompt_ms'],decodeMs=timing['predicted_ms'],inputTokens=row['response']['usage']['prompt_tokens'],outputTokens=row['response']['usage']['completion_tokens'])
                cases.append(dict(reviewCaveat=item.get('reviewCaveat'),reviewQualifier=item.get('reviewQualifier'),modelSource=row['encoding']['cleanedSource'] if definition['id'] in ['private-spans-v1','data-object-v1','2b-precision-v1'] else None,speechCleanup=row['encoding']['cleanup'] if definition['id'] in ['private-spans-v1','data-object-v1','2b-precision-v1'] else None,actualMessages=row['request']['messages'] if definition['id'] in ['current-task-v1','system-task-probe-v1','private-spans-v1','data-object-v1','2b-precision-v1'] else None,wrapper=definition['id']+'/'+row[keyfield] if definition['id'] in ['current-task-v1','system-task-probe-v1','private-spans-v1','data-object-v1','2b-precision-v1'] else None,rawEncoded=row['response']['choices'][0]['message']['content'] if mode=='protected' and row['encoding']['protectedTerms'] else None,encoding=row.get('encoding'),rawPlan=row['response']['choices'][0]['message']['content'] if mode=='sourceplan' and row['variant']=='plan' else None,preparedPlan=row.get('preparedPlan'),renderError=row.get('renderError'),pronunciationEvidence=row.get('pronunciationEvidence'),preparationMs=row.get('preparationMs'),textStageMs=row.get('textStageMs'),role=row.get('role','measured'),model=model,performance=performance,delivery=delivery,round=definition['id'],run=relative,variant=row[keyfield],id=row['id'],cohort=row.get('cohort',row.get('kind','parity' if mode=='parity' else 'development')),scene=scene,effectiveScene=effective,sceneEvidence=evidence,assignment='测试用例指定，未测自动命中',template=read(ROOT/'benchmarks/reproduction'/('rewrite-'+definition['id'] if definition['id'] in ['local-protection-stress-v1','current-task-v1','system-task-probe-v1','private-spans-v1','data-object-v1','2b-precision-v1'] else 'rewrite-local-protection-v1')/'templates'/(scene+'.json'))['version'] if mode=='protected' else ('source-plan-v1' if row['variant']=='plan' else read(ROOT/'benchmarks/reproduction/rewrite-source-plan-v1/templates'/('baseline-'+scene+'.json'))['version']) if mode=='sourceplan' else read(gguf_template)['version'] if mode=='gguf' else row['templateVersion'] if mode=='staged' else row['response']['template'],source=row['source'],context=row.get('context',{}),output=raw_output,cleanup=row['response'].get('cleanup',{}).get('text'),faithful=faithful,useful=useful,issues=item.get('issues',[]),note=item['note'],strict='未启用' if mode in ['gguf','sourceplan','protected'] else row.get('strict','未单独计分'),envelope='未启用' if mode in ['gguf','sourceplan','protected'] else row.get('envelope','未单独计分'),milliseconds=row.get('generationMs')))
                if mode=='staged' and row['stage']=='selection':
                    cases[-1].update(selectionConforms=item['choiceConforms'] if 'choiceConforms' in item else item['faithful'],selectionReviewBasis='explicit choiceConforms' if 'choiceConforms' in item else 'legacy faithful field denotes selection conformance',recordedFaithful=item['faithful'])
                if mode=='staged':
                    cases[-1].update(sourcePreparation=row.get('sourcePreparation'),cleanupObservation=row.get('cleanupObservation'),cleanupIpcMs=row.get('cleanupIpcMs'),policyRoute=row.get('encoding',{}).get('policyRoute'),sourceControl=row.get('sourceControl'),sourceLimited=item.get('sourceLimited'),languagePreserved=item.get('languagePreserved'),evaluationSet=row.get('evaluationSet'),stage=row['stage'],actualMessages=row['request']['messages'],modelSource=row['editedSource'] if row['stage']=='format' else row['cleanup']['text'],speechCleanup=row['cleanup'],wrapper=definition['id']+'/'+row['stage'],rawEncoded=row['response']['choices'][0]['message']['content'] if row['stage']=='format' and row['encoding']['protectedTerms'] else None,template=row['templateVersion'],selectionReference=row.get('selectionReference'),selectionRowSha256=row.get('selectionRowSha256'),actualContextUsed=row['actualContextUsed'],spellingHypotheses=row.get('prepared'),spellingSelection=row.get('selection'),spellingEdits=row['edits'],editedSource=row['editedSource'],pipelineMs=row.get('pipelineMs'),strict='未启用',envelope='未启用',delivery=dict(text=raw_output,origin='两阶段最终正文（不做输出拒绝）',faithful=faithful,note=item['note'],policy='none') if row['stage']=='format' else None)
                if mode=='staged' and row['stage']=='format' and row.get('pipeline') in ['formatter-v3','metal-preflight-v1']:
                    cases[-1]['delivery']['origin']='复用纠错结果后的排版正文（不做输出拒绝）' if row['pipeline']=='formatter-v3' else '单次Metal参数排版探针（不做输出拒绝）'
                if mode=='staged' and row['stage']=='format' and definition['id'] in ['metal-regression-v1','natural-format-v1','language-profile-v1','language-profile-v2']:
                    cases[-1]['delivery']['origin']='本轮排版正文（无新增选择调用，不做输出拒绝）'
                if mode=='staged' and row['stage']=='format' and definition['id'] in ['metal-chain-v1','metal-prefix-v1','metal-primed-v1','history-chain-v1']:
                    cases[-1]['delivery']['origin']='实际清理与按需纠错后的正文（不做输出拒绝）'
                if definition['id']=='history-format-v2':
                    cases[-1]['delivery']['origin']='复用历史实际清理／选择后的新排版正文（不做输出拒绝）'
                if definition['id'] in ['work-scope-v3','work-scope-v4']:
                    cases[-1]['delivery']['origin']='本轮工作排版正文（历史前置阶段复用；构造对照未执行清理，不做输出拒绝）'
                if definition['id'] in ['tone-chain-v1','selector-semantic-v1','selector-semantic-v2','selector-preview-v1'] and row['stage']=='format':
                    cases[-1]['delivery']['origin']='本轮真实清理／按需选择后的最终正文（不做输出拒绝）'
                    if row['selectionRowSha256'] is not None:
                        selected=next(r for r in rows if r['stage']=='selection' and rowhash(r)==row['selectionRowSha256'])
                        cases[-1]['spellingSelection']=selected['selection']
                        if definition['id']=='selector-preview-v1':cases[-1]['candidatePreviews']=json.loads(selected['request']['messages'][-1]['content'])['hypotheses']
                        cases[-1]['selectionReference']=dict(path=str(paths[0].relative_to(ROOT)),variant=selected['variant'],id=selected['id'],rowSha256=rowhash(selected))
                if definition['id']=='selector-semantic-v1' and row['stage']=='selection' and row['arm']=='candidate':
                    cases[-1]['template']='selector-semantic-v1-selection'
                    cases[-1]['note']+=' 原始行误记旧模板身份；本视图按已核验的实际请求显示新模板，原始标签与修正绑定保留。'
                if definition['id'] in ['selector-demo-order-v1','selector-lfm25-v1','selector-youtu-v1','selector-minicpm-f16-v1','selector-minicpm-v1','selector-contrast-demo-v1','selector-unique-label-v1','selector-reference-turn-v1','selector-screen-v1','selector-model-cpu-v1','selector-referent-v1','selector-source-last-v1','selector-sentence-choice-v1','selector-hybrid-v1','instruction-language-qwen-v1','instruction-language-hybrid-v1']:
                    cases[-1]['selectorData']=json.loads(row['request']['messages'][-1]['content'])
                if definition['id']=='selector-demo-order-v1':
                    cases[-1]['demoPairOrder']=[0,1,2,3,7,8,9,4,5,6] if row['arm'] in ['negative-last','negative-last-balanced'] else list(range(10))
                    cases[-1]['nativeBudget']=dict(promptTokens=row['response']['usage']['prompt_tokens'],outputReserve=128,contextLimit=4096,preflightModelGenerations=0,reference='benchmarks/runs/rewrite-selector-demo-order-v1/preflight/requests.jsonl')
                if definition['id']=='selector-contrast-demo-v1':
                    cases[-1]['addedContrastDemoMessages']=8 if row['arm'] in ['contrast-demo','contrast-demo-balanced'] else 0
                    cases[-1]['nativeBudget']=dict(promptTokens=row['response']['usage']['prompt_tokens'],outputReserve=128,contextLimit=4096,preflightModelGenerations=0,reference='benchmarks/runs/rewrite-selector-contrast-demo-v1/preflight/requests.jsonl')
                if definition['id']=='selector-unique-label-v1':
                    cases[-1]['uniqueLabelsOmitted']=sum('status' not in h for message in row['request']['messages'] if message['role']=='user' for h in json.loads(message['content'])['hypotheses'])
                if definition['id']=='selector-reference-turn-v1':
                    cases[-1]['referenceMessageRoles']=[message['role'] for message in row['request']['messages']]
                    cases[-1]['runtimeConfiguration']=dict(nativeCommand=next(w for w in manifest['workers'] if w['arm']==row['arm'] and w['scene']==row['scene'])['command'],sourcePreparationsStudy=457,formatterExecuted=False,toolExecution=False,templatePreflightGenerations=0)
                if definition['id'] in ['instruction-language-qwen-v1','instruction-language-hybrid-v1']:
                    cases[-1].update(trustedInstructionLanguage='en',referenceInstructionLanguage='zh')
                if definition['id']=='selector-sentence-choice-v1':
                    cases[-1].update(normalizedSpellingSelection=item['normalizedChoices'],sentenceOriginalLabel='A' if row['arm']=='sentence-ab' else 'B' if row['arm']=='sentence-ba' else None,choiceEnums={key:field['enum'] for key,field in row['request']['response_format']['schema']['properties'].items()})
                if definition['id']=='selector-source-last-v1':
                    cases[-1]['sourceFieldOrder']=[list(json.loads(message['content'])) for message in row['request']['messages'] if message['role']=='user']
                if definition['id'] in ['selector-lfm25-v1','selector-youtu-v1','selector-minicpm-f16-v1','selector-minicpm-v1','selector-model-cpu-v1','selector-hybrid-v1','instruction-language-qwen-v1','instruction-language-hybrid-v1']:
                    actual_worker=next(w for w in manifest['workers'] if w['arm']==row['arm'] and w['scene']==row['scene'])
                    cases[-1]['runtimeConfiguration']=dict(nativeCommand=actual_worker['command'],comparisonReference=row['comparisonReference'],newGoPreparations=0,formatterExecuted=False)
                if definition['id']=='selector-referent-v1':
                    cases[-1].update(normalizedSpellingSelection=item['normalizedChoices'],descriptionReview=item['descriptionReview'],descriptionConforms=item['descriptionConforms'],descriptionOrderConforms=item['descriptionOrderConforms'],descriptionVerdict=None if item['descriptionReview'] is None else '含义核对通过' if item['descriptionConforms'] is True else '含义核对未通过' if item['descriptionConforms'] is False else '含义核对待核验')
                if definition['id']=='selector-preview-v1' and row['stage']=='selection':
                    cases[-1]['candidatePreviews']=json.loads(row['request']['messages'][-1]['content'])['hypotheses']
                if mode=='staged' and 'templatePath' in row:
                    cases[-1]['promptProvenance']={k:row[k] for k in ['templatePath','templateSha256','actualMessagesSha256']}
                entry['count']+=1
            sources += [dict(path=str(p.relative_to(ROOT)),sha256=digest(p)) for p in paths]
        rounds.append(entry)
    for path in [ROOT/'benchmarks/runs/rewrite-local-protection-v1/host-observation.json',ROOT/'benchmarks/runs/rewrite-local-protection-v1/granite-template-diagnostic.json',ROOT/'benchmarks/reproduction/rewrite-local-protection-v1/freeze.json',ROOT/'benchmarks/reproduction/rewrite-local-protection-stress-v1/freeze.json',ROOT/'benchmarks/artifacts/cmudict-pronunciation-v1/provenance.json',ROOT/'benchmarks/reproduction/rewrite-pronunciation-evidence-v1/protocol-clarification.json',ROOT/'benchmarks/runs/rewrite-source-plan-v1/host-observation.json',ROOT/'benchmarks/reproduction/rewrite-source-plan-v1/freeze.json',ROOT/'benchmarks/reproduction/rewrite-source-plan-v1/protocol.json',ROOT/'benchmarks/runs/rewrite-2b-contrast-v1/host-provenance.json',ROOT/'benchmarks/runs/rewrite-pronunciation-evidence-v1/host-provenance.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for base,names in [('benchmarks/corpora/rewrite-history-v1',['manifest.json','lineage.json']),('benchmarks/reproduction/rewrite-history-chain-v1',['freeze.json','protocol.json']),('benchmarks/runs/rewrite-history-chain-v1/run',['selection.json','case-outcomes.jsonl','review-amendment.json','review-v1.json','producer-coverage.json'])]:
        for name in names:
            path=ROOT/base/name;sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-history-format-v2'/name for name in ['freeze.json','protocol.json','prompts.json']]+[ROOT/'benchmarks/runs/rewrite-history-format-v2/run/selection.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for study_name in ['rewrite-work-scope-v3','rewrite-work-scope-v4']:
        for path in [ROOT/'benchmarks/reproduction'/study_name/name for name in ['freeze.json','protocol.json','prompts.json']]+[ROOT/'benchmarks/runs'/study_name/'run/selection.json']:
            sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/corpora/rewrite-work-scope-v3/manifest.json',ROOT/'benchmarks/runs/rewrite-work-scope-v4/run/unchanged-scene-request-parity.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-tone-chain-v1'/name for name in ['freeze.json','protocol.json']]+[ROOT/'benchmarks/runs/rewrite-tone-chain-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','reference-parity.json','reference-recovery-observation.json','review-amendment.json','review-initial.json']]+[ROOT/'benchmarks/runs/rewrite-tone-chain-v1/producer-audit.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-semantic-v1'/name for name in ['freeze.json','protocol.json','prompts.json']]+[ROOT/'benchmarks/runs/rewrite-selector-semantic-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','selector-template-provenance.json','reference-comparison.json','review-amendment.json','review-initial.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-semantic-v2'/name for name in ['freeze.json','protocol.json','prompts.json']]+[ROOT/'benchmarks/runs/rewrite-selector-semantic-v2/run'/name for name in ['selection.json','case-outcomes.jsonl','reference-comparison.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-preview-v1'/name for name in ['freeze.json','protocol.json','prompts.json','preview.py']]+[ROOT/'benchmarks/runs/rewrite-selector-preview-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','reference-comparison.json','supplementary-observations.json']]+[ROOT/'benchmarks/runs/rewrite-selector-preview-v1/midrun-load-observation.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-screen-v1'/name for name in ['freeze.json','protocol.json','comparison.py']]+[ROOT/'benchmarks/reproduction/rewrite-selector-screen-v1/templates'/arm/'prompts.json' for arm in ['baseline','after','balanced','role']]+[ROOT/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json']+[ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run'/name for name in ['selection.json','preparations.jsonl','case-outcomes.jsonl','reference-comparison.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in (ROOT/'benchmarks/runs/rewrite-selector-model-v1').rglob('*'):
        if path.is_file() and path.suffix in ['.json','.jsonl','.log']:
            sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-model-cpu-v1'/name for name in ['freeze.json','protocol.json','run.py']]+[ROOT/'benchmarks/runs/rewrite-selector-model-cpu-v1/run'/name for name in ['selection.json','case-outcomes.jsonl']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-referent-v1'/name for name in ['freeze.json','protocol.json','prompts.json','pipeline.py']]+[ROOT/'benchmarks/runs/rewrite-selector-referent-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-budget-comparison.json','description-reviews.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-source-last-v1'/name for name in ['freeze.json','protocol.json','layout.json','pipeline.py']]+[ROOT/'benchmarks/runs/rewrite-selector-source-last-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-parity.json','input-order-comparison.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-sentence-choice-v1'/name for name in ['freeze.json','protocol.json','prompts.json','pipeline.py']]+[ROOT/'benchmarks/runs/rewrite-selector-sentence-choice-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-parity.json','counterbalance-comparison.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-hybrid-v1'/name for name in ['freeze.json','protocol.json','preflight-plan.json']]+[ROOT/'benchmarks/runs/rewrite-selector-hybrid-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','paired-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-hybrid-v1/preflight'/name for name in ['manifest.json','results.jsonl','review.json','qualification.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-instruction-language-v1'/name for name in ['freeze.json','protocol.json','prompts.json','runtime-templates.json']]+[ROOT/'benchmarks/runs/rewrite-selector-instruction-language-v1'/key/name for key in ['qwen','hybrid'] for name in ['selection.json','case-outcomes.jsonl','paired-comparison.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-reference-turn-v1'/name for name in ['freeze.json','protocol.json','layout.json','pipeline.py']]+[ROOT/'benchmarks/runs/rewrite-selector-reference-turn-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-parity.json','reference-turn-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-reference-turn-v1/preflight'/name for name in ['serialization.json','status.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-unique-label-v1'/name for name in ['freeze.json','protocol.json','layout.json','pipeline.py']]+[ROOT/'benchmarks/runs/rewrite-selector-unique-label-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-parity.json','unique-label-comparison.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-contrast-demo-v1'/name for name in ['freeze.json','protocol.json','layout.json','pipeline.py','demonstrations.json']]+[ROOT/'benchmarks/runs/rewrite-selector-contrast-demo-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-parity.json','contrast-demo-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-contrast-demo-v1/preflight'/name for name in ['status.json','requests.jsonl']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-minicpm-v1'/name for name in ['freeze.json','protocol.json','preflight-plan.json','comparison-addendum.json','sources/native-template-policy.json']]+[ROOT/'benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run'/name for name in ['selection.json','case-outcomes.jsonl','paired-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-minicpm-v1/preflight'/name for name in ['manifest.json','results.jsonl','review.json','qualification.json']]+[ROOT/'benchmarks/runs/rewrite-selector-minicpm-v1/aborted-before-qualification'/name for name in ['closure.json','terminal-status.json','run/manifest.json','run/results.jsonl','run/selector-model-v1-baseline-work-server.log']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-minicpm-f16-v1'/name for name in ['freeze.json','protocol.json','preflight-plan.json','sources/native-template-policy.json','sources/artifact-comparison.json','sources/complete-metadata-identity.json']]+[ROOT/'benchmarks/runs/rewrite-selector-minicpm-f16-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','paired-comparison.json','q4-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-minicpm-f16-v1/preflight'/name for name in ['manifest.json','results.jsonl','review.json','qualification.json']]+[ROOT/'benchmarks/runs/rewrite-minicpm-tokenizer-v1'/name for name in ['plan.json','status.json','native/tokenization.jsonl']]+[ROOT/'benchmarks/reproduction/rewrite-minicpm-tokenizer-v1'/name for name in ['tokenizer.json','receipt.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-youtu-v1'/name for name in ['freeze.json','comparison-sources.json','protocol.json','preflight-plan.json','sources/native-template-policy.json','sources/artifact-qualification.json','sources/license-research-notes.json']]+[ROOT/'benchmarks/runs/rewrite-selector-youtu-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','paired-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-youtu-v1/preflight'/name for name in ['manifest.json','results.jsonl','review.json','qualification.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-lfm25-v1'/name for name in ['freeze.json','comparison-sources.json','protocol.json','preflight-plan.json','sources/native-template-policy.json','sources/native-log-encoding.json','sources/artifact-qualification.json','sources/render-comparison-v2/summary.json','sources/render-comparison-v2/prompts.jsonl']]+[ROOT/'benchmarks/runs/rewrite-selector-lfm25-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','paired-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-lfm25-v1/preflight'/name for name in ['manifest.json','results.jsonl','review.json','qualification.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-selector-demo-order-v1'/name for name in ['freeze.json','hypothesis.json','protocol.json','layout.json','pipeline.py','test-erratum.json','test_report_v2.py']]+[ROOT/'benchmarks/runs/rewrite-selector-demo-order-v1/run'/name for name in ['selection.json','case-outcomes.jsonl','control-parity.json','demo-order-comparison.json']]+[ROOT/'benchmarks/runs/rewrite-selector-demo-order-v1/preflight'/name for name in ['status.json','requests.jsonl']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    sources+=d1_summary['sources']
    for path in [ROOT/'benchmarks/runs/rewrite-d1-evidence-v1'/name for name in ['review.json','selection.json','manual-review-completion.json']]+[ROOT/'benchmarks/runs/rewrite-selector-d1-policy-v1/budget'/name for name in ['summary.json','requests.jsonl','native-calls.jsonl']]+[ROOT/'benchmarks/reproduction/rewrite-selector-d1-policy-v1'/name for name in ['identity-clarification.json','sources/server-decision.cpp','layout.py','protocol.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    failure_root=ROOT/'benchmarks/runs/rewrite-small-cache-parity-v1'
    sources += [dict(path=str(path.relative_to(ROOT)),sha256=digest(path)) for path in [failure_root/'failed/manifest.json',failure_root/'failure.json',failure_root/'run-environment.json',failure_root/'failed/forward-uncached-work-server.log']]
    for path in [ROOT/'benchmarks/corpora/rewrite-natural-asr-v1/source-audit.json',ROOT/'benchmarks/corpora/rewrite-natural-asr-v1/inputs.json',ROOT/'benchmarks/corpora/rewrite-natural-asr-v1/evaluator.json',ROOT/'benchmarks/reproduction/rewrite-timing-audit-v1/consistency.json',ROOT/'benchmarks/reproduction/rewrite-timing-audit-v1/clock-calibration.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for study in ['rewrite-current-task-v1','rewrite-system-task-probe-v1']:
        for path in [ROOT/'benchmarks/reproduction'/study/'freeze.json',ROOT/'benchmarks/reproduction'/study/'cleanup-clarification.json',ROOT/'benchmarks/runs'/study/'selection.json']:
            sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/runs/rewrite-cleanup-preflight-v1/provenance.json',ROOT/'benchmarks/runs/rewrite-cleanup-preflight-v1/observations.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/runs/rewrite-private-spans-v1/selection.json',ROOT/'benchmarks/runs/rewrite-private-spans-v1/host-observation.json',ROOT/'benchmarks/reproduction/rewrite-private-spans-v1/freeze.json',ROOT/'benchmarks/reproduction/rewrite-private-spans-v1/packaging-clarification.json',ROOT/'benchmarks/runs/rewrite-natural-text-review-v1/manifest.json',ROOT/'benchmarks/runs/rewrite-natural-text-review-v1/review.json',ROOT/'benchmarks/runs/rewrite-natural-text-review-v1/cleanup-execution.json',ROOT/'benchmarks/runs/rewrite-natural-text-review-v1/cleanup-observations.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/runs/rewrite-data-object-v1/selection.json',ROOT/'benchmarks/reproduction/rewrite-data-object-v1/freeze.json',ROOT/'benchmarks/reproduction/rewrite-data-object-v1/protocol.json',ROOT/'benchmarks/reproduction/rewrite-data-object-v1/execution-clarification.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/runs/rewrite-2b-precision-v1/selection.json',ROOT/'benchmarks/runs/rewrite-2b-precision-v1/host-observation.json',ROOT/'benchmarks/reproduction/rewrite-2b-precision-v1/freeze.json',ROOT/'benchmarks/reproduction/rewrite-2b-precision-v1/protocol.json',ROOT/'benchmarks/reproduction/rewrite-2b-precision-v1/acquisition.json',ROOT/'benchmarks/reproduction/rewrite-2b-precision-v1/gguf-info.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/runs/rewrite-data-object-v1/report-loader-failure-v1/manifest.json',ROOT/'benchmarks/runs/rewrite-data-object-v1/report-loader-fix.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-bounded-edit-v1/freeze.json',ROOT/'benchmarks/reproduction/rewrite-bounded-edit-v1/protocol.json',ROOT/'benchmarks/runs/rewrite-bounded-edit-v1/run/selection.json',ROOT/'benchmarks/runs/rewrite-bounded-edit-v1/producer-language-audit.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-bounded-edit-v2/freeze.json',ROOT/'benchmarks/reproduction/rewrite-bounded-edit-v2/protocol.json',ROOT/'benchmarks/runs/rewrite-bounded-edit-v2/run/selection.json',ROOT/'benchmarks/runs/rewrite-bounded-edit-v2/host-provenance.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-formatter-v3/freeze.json',ROOT/'benchmarks/reproduction/rewrite-formatter-v3/protocol.json',ROOT/'benchmarks/runs/rewrite-formatter-v3/run/selection.json',ROOT/'benchmarks/artifacts/rewrite-model-refresh-v1/provenance.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-metal-preflight-v1/freeze.json',ROOT/'benchmarks/runs/rewrite-metal-preflight-v1/load-observation/manifest.json',ROOT/'benchmarks/runs/rewrite-metal-preflight-v1/load-observation/load.log']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for study in ['rewrite-metal-prefix-v1','rewrite-metal-chain-v1','rewrite-metal-regression-v1','rewrite-natural-format-v1','rewrite-language-profile-v1','rewrite-language-profile-v2']:
        for path in [ROOT/'benchmarks/reproduction'/study/'freeze.json',ROOT/'benchmarks/reproduction'/study/'protocol.json',ROOT/'benchmarks/runs'/study/'run/selection.json']:
            sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/runs/rewrite-cleanup-copula-v1/manifest.json',ROOT/'benchmarks/runs/rewrite-cleanup-copula-v1/observations.json']:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    for path in [ROOT/'benchmarks/reproduction/rewrite-metal-primed-v1'/name for name in ['freeze.json','protocol.json','warmups.json']]:
        sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    pairing=ROOT/'benchmarks/runs/rewrite-metal-prefix-v1/run/paired-controls.json'
    sources.append(dict(path=str(pairing.relative_to(ROOT)),sha256=digest(pairing)))
    historical=ROOT/'benchmarks/runs/scene-rewrite-alignment-v1/report/summary.json'
    sources.append(dict(path=str(historical.relative_to(ROOT)),sha256=digest(historical)))
    audit_path=ROOT/'benchmarks/runs/rewrite-retrospective-v1/audit-review.jsonl'
    audit_manifest=read(audit_path.with_name('audit-manifest.json'))
    if audit_manifest['status']!='complete' or audit_manifest['reviewSha256']!=digest(audit_path):raise ValueError('Changed audit review')
    originals={}
    for source in audit_manifest['sources']:
        path=ROOT/source['path']
        if digest(path)!=source['sha256']:raise ValueError('Changed original audit')
        original_rows=read(path) if path.suffix=='.json' else [json.loads(x) for x in path.read_text().splitlines()]
        for row in original_rows:originals[(source['path'],rowhash(row))]=row
        sources.append(source)
    audits=[json.loads(x) for x in audit_path.read_text().splitlines()]
    if len(audits)!=113 or len({(r['phase'],r['variant'],r['id']) for r in audits})!=113:raise ValueError('Incomplete audit inspection')
    for item in audits:
        original=originals[(item['originalPath'],item['originalRowSha256'])]
        if item['output']!=original['audit']['response']['text']:raise ValueError('Changed audit output')
        pair=original['pair'] if 'pair' in original else json.loads(original['audit']['response']['cleanup']['text'])
        if item['source']!=pair['source'] or item['candidate']!=pair['candidate']:raise ValueError('Changed audited input')
    sources += [dict(path=str(p.relative_to(ROOT)),sha256=digest(p)) for p in [audit_path,audit_path.with_name('audit-manifest.json')]]
    corpus_path=ROOT/'benchmarks/corpora/entity-ocr-v1/corpus.json'
    observations_path=ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/ocr/observations.jsonl'
    observations=[json.loads(x) for x in observations_path.read_text().splitlines()]
    fixtures=[]
    for sample in read(corpus_path)['samples']:
        img=corpus_path.parent/sample['contextFixture']['image']
        fixtures.append(dict(id=sample['id'],source=sample['source'],truth=sample['contextFixture']['screenTruth'],image='data:image/png;base64,'+base64.b64encode(img.read_bytes()).decode(),observations=[r for r in observations if r['caseId']==sample['id']]))
    sources += [dict(path=str(p.relative_to(ROOT)),sha256=digest(p)) for p in [corpus_path,observations_path,ROOT/'benchmarks/reproduction/rewrite-entity-evidence-v1/freeze.json',ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/path-mappings.json',ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/ocr/manifest.json',ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/ocr/metrics.json',ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/ocr/host-provenance.json']]
    alignment_module=load_report('rewrite-spelling-alignment-v1','spelling_alignment','review.py')
    alignment_manifest=read(ROOT/'benchmarks/runs/rewrite-spelling-alignment-v1/manifest.json')
    aligned_runs={str(pathlib.PurePosixPath(name).parent.relative_to('benchmarks/runs')) for name in alignment_manifest['files'] if name.endswith('/results.jsonl')}
    spelling_alignment=alignment_module.apply([c for c in cases if c['run'] in aligned_runs])
    for entry in rounds:
        relevant=[c for c in cases if c['round']==entry['id']]
        if any(c.get('alignedReview') for c in relevant):
            variants=sorted({c['variant'] for c in relevant})
            entry['alignmentSummary']=[dict(variant=v,total=sum(c['variant']==v for c in relevant),originalFaithful=sum(c['variant']==v and c['faithful'] for c in relevant),alignedFaithful=sum(c['variant']==v and c.get('alignedReview',{}).get('faithful',c['faithful']) for c in relevant)) for v in variants]
    for name in ['manifest.json','review.json','policy.json']:
        path=ROOT/'benchmarks/runs/rewrite-spelling-alignment-v1'/name;sources.append(dict(path=str(path.relative_to(ROOT)),sha256=digest(path)))
    return dict(nativeD1=d1_summary,selectorDemoOrder=demo_order_summary,selectorLfm25=lfm25_summary,selectorYoutu=youtu_summary,selectorMinicpmF16=minicpm_f16_summary,minicpmTokenizerParity=minicpm_tokenizer,selectorMinicpm=minicpm_summary,selectorContrastDemo=contrast_demo_summary,selectorUniqueLabel=unique_label_summary,selectorReferenceTurn=reference_turn_summary,instructionLanguage=instruction_language_summary,selectorHybrid=hybrid_summary,selectorSentenceChoice=sentence_choice_summary,selectorSourceLast=source_last_summary,selectorReferent=referent_summary,selectorModelCpu=cpu_model_summary,runtimeDiagnostics=runtime_diagnostics,selectorScreen=screen_summary,selectorPreview=preview_summary,selectorSemanticV2=semantic_v2_summary,selectorSemantic=semantic_summary,toneChain=tone_summary,workScope=dict(v3=work_v3_summary,v4=work_v4_summary),historyFormatter=history_format_summary,historyCoverage=history_summary,priming=primed_summary,cleanupCopula=dict(observations=copula['observations'],version=copula['version'],newLlmGenerations=0,changedCases=copula['changedCases']),prefixCache=dict(generations=174,rawParity=174,cacheHits=164,cachedTokens=90542,p50DevelopmentMs=694.2060419823974,p95DevelopmentMs=2419.5817501749843),continuousChain=dict(generations=174,selectorCalls=18,finalBodies=156,developmentFaithful=36,developmentUseful=36,p95DevelopmentMs=2921.6538751497865),naturalCandidateReview=dict(sources=120,generations=1560,latest='selector-preview-v1',faithful=119,uncertain=1,languagePreserved=120),spellingAlignment=spelling_alignment,naturalTextReview=dict(reviewed=natural_text['reviewed'],cleanupObservations=120,changedCleanupSources=0,newInference=False),naturalAsrPreparation=dict(sources=natural['eligibleNaturalSources'],languages=natural['languages'],newInference=natural['newInference']),timingIntegrity=dict(observations=timing_audit['observations'],flagged=timing_audit['flagged']),historical='alignment-v1 原始代理通过数为 production/strict 10/24、scene/envelope 18/24。已补完 96 条改写与 113 条语义审计的事后逐条复核，并以原始 SHA-256 绑定。事后判断不回写原指标，也不替代独立盲评。',audits=audits,ocrMetrics=read(ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/ocr/metrics.json'),ocrFixtures=fixtures,rounds=rounds,cases=cases,gates=GATES,sources=sources,scope='人工构造文本与图片、历史自然ASR源的实际候选、实际本地 LLM 和 macOS Vision OCR、助手逐条判断。场景为测试指定；不同轮次不可直接排序。最低验收目标：Apple Silicon／16GB；当前M4／32GB仅为开发机证据，尚未完成最低配置验收。',status='持续迭代中，尚未达到上线门槛')

def encode_data(data):
    return json.dumps(data,ensure_ascii=False).replace('<','\\u003c').replace('>','\\u003e').replace('&','\\u0026')

def main():
    data=collect();OUT.mkdir(parents=True,exist_ok=True)
    encoded=encode_data(data)
    template=(pathlib.Path(__file__).parent/'template.html').read_text()
    document=template.replace('<!-- REPORT_DATA -->',encoded)
    (OUT/'index.html').write_text(document)
    (OUT/'data.json').write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(rounds=len(data['rounds']),reviewedOutputs=len(data['cases']),html=str(OUT/'index.html')),ensure_ascii=False))
if __name__=='__main__':main()
