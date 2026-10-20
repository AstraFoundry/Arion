# Retrospective manual review

No new inference. Original selection and proxy scores remain unchanged. Assistant inspection under the current rewrite contract; not independent blind ratings.

## dev / production / dev-review-en

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
Hi Ellis, the report draft is attached, please review it before Tuesday and contact me with any questions, thanks.
```

**Raw output**

```text
Hi Ellis,

The report draft is attached. Please review it before Tuesday and contact me with any questions.

Thanks.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Ellis, the report draft is attached, please review it before Tuesday and contact me with any questions, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-pagination-zh

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
给 /api/orders 加分页，默认每页二十条，保留原来的排序，不要改返回字段。
```

**Raw output**

```text
给 /api/orders 加分页，默认每页二十条，保留原来的排序，不要改返回字段。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
给 /api/orders 加分页，默认每页二十条，保留原来的排序，不要改返回字段。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-delay-en

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
Hi Jordan, the samples have not arrived, they are expected Friday, and I will schedule testing after they arrive, thanks.
```

**Raw output**

```text
Hi Jordan,

The samples have not arrived. They are expected to arrive Friday. I will schedule testing after they arrive. Thanks.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Jordan, the samples have not arrived, they are expected Friday, and I will schedule testing after they arrive, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-uncertain-en

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
I may arrive around six but I am not sure yet, please do not order food for me.
```

**Raw output**

```text
I may arrive around six but I am not sure yet, please do not order food for me.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
I may arrive around six but I am not sure yet, please do not order food for me.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-cache-zh

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
给 getPrices 加 Redis 缓存，过期时间五分钟，价格变动时失效，不要缓存错误。
```

**Raw output**

```text
给 `getPrices` 加 Redis 缓存，过期时间五分钟，价格变动时失效，不要缓存错误。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
给 getPrices 加 Redis 缓存，过期时间五分钟，价格变动时失效，不要缓存错误。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-cache-en

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
Add Redis caching to getPrices with a five minute TTL, invalidate it when prices change and do not cache errors.
```

**Raw output**

```text
Add Redis caching to getPrices with a five minute TTL, invalidate it when prices change and do not cache errors.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Add Redis caching to getPrices with a five minute TTL, invalidate it when prices change and do not cache errors.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-leave-zh

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Raw output**

```text
周经理您好：

我周四请假一天，交接给小赵了。麻烦批准，谢谢。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-payment-zh

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
支付接口做好了，测试环境还没准备好，等环境好了再联调，今天先别发布。
```

**Raw output**

```text
支付接口做好了，测试环境还没准备好，等环境好了再联调，今天先别发布。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
支付接口做好了，测试环境还没准备好，等环境好了再联调，今天先别发布。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-pagination-en

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
Add pagination to /api/orders with twenty items per page, keep the current ordering and do not change response fields.
```

**Raw output**

```text
Add pagination to /api/orders with twenty items per page, keep the current ordering, do not change response fields.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Add pagination to /api/orders with twenty items per page, keep the current ordering and do not change response fields.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-question-en

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
Can you check why this endpoint is so slow?
```

**Raw output**

```text
Can you check why this endpoint is so slow?
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Can you check why this endpoint is so slow?
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-owners-zh

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Raw output**

```text
小陈负责设计，小刘负责测试。设计稿周三交，测试周五完成。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-owners-en

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
Mara owns design and Leo owns testing, the design is due Wednesday and testing finishes Friday.
```

**Raw output**

```text
Mara owns design and Leo owns testing, the design is due Wednesday and testing finishes Friday.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Mara owns design and Leo owns testing, the design is due Wednesday and testing finishes Friday.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-risk-zh

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
目前可能是缓存的问题，还没有确认，先看日志，不要直接清理生产缓存。
```

**Raw output**

```text
目前可能是缓存的问题，还没有确认，先看日志，不要直接清理生产缓存。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
目前可能是缓存的问题，还没有确认，先看日志，不要直接清理生产缓存。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-repair-en

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
Let us meet Wednesday, actually Thursday afternoon, at the usual place.
```

**Raw output**

```text
Let us meet Wednesday, actually Thursday afternoon, at the usual place.
```

All inspected source facts, dates, roles, restrictions and request intent retained. Spoken Wednesday/actually-Thursday correction remains.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Let us meet Wednesday, actually Thursday afternoon, at the usual place.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-investigate-zh

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
分析 OrderList.tsx 里的 TS2345 报错，先解释原因，不要提交代码。
```

**Raw output**

```text
TS2345 报错通常表示 TypeScript 编译器在解析代码时遇到了问题。这种错误通常出现在以下几种情况：

1. **类型不匹配**：在定义变量或函数时，类型不匹配，导致 TS2345 报错。

2. **变量未声明**：在使用变量之前，未声明该变量，导致 TS2345 报错。

3. **函数未声明**：在定义函数之前，未声明该函数，导致 TS2345 报错。

4. **类未声明**：在定义类
```

Invents an explanation of TS2345 instead of preserving the request; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
分析 OrderList.tsx 里的 TS2345 报错，先解释原因，不要提交代码。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-investigate-en

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
Investigate TS2345 in OrderList.tsx, explain the cause first, and do not submit code.
```

**Raw output**

```text
Investigate TS2345 in OrderList.tsx, explain the cause first, and do not submit code.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Investigate TS2345 in OrderList.tsx, explain the cause first, and do not submit code.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-question-zh

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
你能帮我看看这个接口为什么这么慢吗？
```

**Raw output**

```text
你能帮我看看这个接口为什么这么慢吗？
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
你能帮我看看这个接口为什么这么慢吗？
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-repair-zh

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
我们约周三，不对，改成周四下午，地点还是老地方。
```

**Raw output**

```text
我们约周四下午，地点还是老地方。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
我们约周三，不对，改成周四下午，地点还是老地方。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-risk-en

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
This might be a cache issue but it is not confirmed, check the logs first and do not clear the production cache.
```

**Raw output**

```text
This might be a cache issue, but it is not confirmed. Check the logs first and do not clear the production cache.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
This might be a cache issue but it is not confirmed, check the logs first and do not clear the production cache.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-review-zh

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
李老师您好，报告初稿已附上，麻烦周二之前看一下，有问题随时联系，谢谢。
```

**Raw output**

```text
李老师您好：

报告初稿已附上。麻烦您在周二之前看一下，如有问题随时联系。

谢谢。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
李老师您好，报告初稿已附上，麻烦周二之前看一下，有问题随时联系，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-delay-zh

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
王先生您好，样品还没到，预计周五到货，收到后我再安排测试，谢谢。
```

**Raw output**

```text
王先生您好：

样品还没到，预计周五到货。收到后我再安排测试。

谢谢。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
王先生您好，样品还没到，预计周五到货，收到后我再安排测试，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-leave-en

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
Hi Rowan, I need Thursday off, I handed my work to Priya, please approve it, thanks.
```

**Raw output**

```text
Hi Rowan,

I need Thursday off.

I handed my work to Priya.

Please approve it.

Thanks.
```

All inspected source facts, dates, roles, restrictions and request intent retained. Each body sentence is a separate paragraph.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Hi Rowan, I need Thursday off, I handed my work to Priya, please approve it, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-payment-en

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
The payment API is finished, the test environment is not ready, start integration when it is ready, and do not release today.
```

**Raw output**

```text
The payment API is finished, the test environment is not ready, start integration when it is ready, and do not release today.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
The payment API is finished, the test environment is not ready, start integration when it is ready, and do not release today.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / production / dev-uncertain-zh

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
我大概六点到，还不能确定，你们先别点我的饭。
```

**Raw output**

```text
我大概六点到，还不能确定，你们先别点我的饭。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
我大概六点到，还不能确定，你们先别点我的饭。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-review-en

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
Hi Ellis, the report draft is attached, please review it before Tuesday and contact me with any questions, thanks.
```

**Raw output**

```text
Hi Ellis,

The report draft is attached. Please review it before Tuesday.

If you have any questions, please contact me.

Thanks.
```

Salutation, attachment status, Tuesday deadline, contact request and thanks retained in natural paragraphs.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Ellis, the report draft is attached, please review it before Tuesday and contact me with any questions, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-pagination-zh

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
给 /api/orders 加分页，默认每页二十条，保留原来的排序，不要改返回字段。
```

**Raw output**

```text
Status: The API endpoint /api/orders has been configured with pagination.  
Pagination:  
- **Page size**: 20  
- **Sorting**: Preserved original sorting.  
Return fields: Unmodified, including return fields.  

**Note**: The actual implementation would involve modifying the API endpoint to include pagination logic, while preserving the original sorting and return fields.
```

Adds completed pagination and implementation explanation, switches language and expands scope.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
给 /api/orders 加分页，默认每页二十条，保留原来的排序，不要改返回字段。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-delay-en

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
Hi Jordan, the samples have not arrived, they are expected Friday, and I will schedule testing after they arrive, thanks.
```

**Raw output**

```text
Hi Jordan,

The samples have not arrived. They are expected to arrive on Friday. I will schedule testing after they arrive.

Thanks.
```

No-arrival status, expected Friday arrival and testing only after arrival all retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Jordan, the samples have not arrived, they are expected Friday, and I will schedule testing after they arrive, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-uncertain-en

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
I may arrive around six but I am not sure yet, please do not order food for me.
```

**Raw output**

```text
I may arrive around six but I am not sure yet, please do not order food for me.
```

Tentative six arrival and no-food-order request preserved.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
I may arrive around six but I am not sure yet, please do not order food for me.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-cache-zh

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
给 getPrices 加 Redis 缓存，过期时间五分钟，价格变动时失效，不要缓存错误。
```

**Raw output**

```text
**修改后的 `getPrices`` 函数：**

```python
def getPrices():
    # 获取当前价格
    current_price = get_current_price()  # 假设此函数返回当前价格

    # 从 Redis 中获取价格
    cached_price = get_redis_price()  # 假设此函数从 Redis 中获取价格

    # 如果缓存中存在价格，且未过期，则使用缓存的价格
    if cached_price is not None and cached_price > 0:
        return cached_price

    # 如果缓存中没有有效价格，则返回当前
```

Implements invented Python cache code instead of rewriting the request; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
给 getPrices 加 Redis 缓存，过期时间五分钟，价格变动时失效，不要缓存错误。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-cache-en

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
Add Redis caching to getPrices with a five minute TTL, invalidate it when prices change and do not cache errors.
```

**Raw output**

```text
**Redis Caching Configuration for `getPrices`**

- **Cache Key:** `getPrices:${product_id}`  
- **TTL:** 5 minutes  
- **Cache Value:** Price data (e.g., `{"price": 100, "timestamp": 163840000}"  
- **Invalidation:** Triggered when the price data changes (e.g., `{"price": 150, "timestamp": 163840000}"  
- **Error Handling:** Do not cache errors (e.g., `{"error
```

Invents cache-key structure, example data and implementation details; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
Add Redis caching to getPrices with a five minute TTL, invalidate it when prices change and do not cache errors.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-leave-zh

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

**Raw output**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Leave date/duration, handoff, approval request and thanks retained; salutation/body remain inline.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-payment-zh

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
支付接口做好了，测试环境还没准备好，等环境好了再联调，今天先别发布。
```

**Raw output**

```text
进展：支付接口已完成并部署。  
待办：测试环境尚未准备好，需等待环境恢复后再进行联调。  
发布要求：今日先不发布。  
备注：今日先进行环境准备，待环境恢复后再进行联调。
```

Invents deployment and extra environment-preparation work; original only says interface finished.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
支付接口做好了，测试环境还没准备好，等环境好了再联调，今天先别发布。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-pagination-en

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
Add pagination to /api/orders with twenty items per page, keep the current ordering and do not change response fields.
```

**Raw output**

```text
Status: Pagination added to /api/orders with 20 items per page.

Next step: Maintain current ordering and do not change response fields.

Delivery: Send only after the review.
```

Changes requested pagination into completed work and adds review-before-delivery condition from a demonstration.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
Add pagination to /api/orders with twenty items per page, keep the current ordering and do not change response fields.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-question-en

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
Can you check why this endpoint is so slow?
```

**Raw output**

```text
The endpoint is slow due to the following reasons:

1. **High Volume of Requests**: The endpoint receives a large number of requests, which can lead to increased processing time.

2. **Resource Intensive Operations**: The endpoint performs operations that are resource-intensive, such as database queries, file I/O, or complex calculations.

3. **Latency in External Services**: The endpoint relies on external services (e.g., APIs, databases, or third-party systems) that have high latency or
```

Answers with speculative reasons for slowness instead of preserving the question; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
Can you check why this endpoint is so slow?
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-owners-zh

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

**Raw output**

```text
进展：小陈负责设计，小刘负责测试。设计稿将于周三提交，测试工作将于周五完成。
```

Both owners/tasks and exact dates retained, but a heading does not group each owner with their task/date.

Faithful: True; scene ready: False; strict: unsupported_content

**Cleanup**

```text
小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-owners-en

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
Mara owns design and Leo owns testing, the design is due Wednesday and testing finishes Friday.
```

**Raw output**

```text
进展：Mara 负责设计，Leo 负责测试。设计截止日期为周三，测试截止日期为周五。
```

Translates the message to Chinese; role/date relationships otherwise preserved.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
Mara owns design and Leo owns testing, the design is due Wednesday and testing finishes Friday.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-risk-zh

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
目前可能是缓存的问题，还没有确认，先看日志，不要直接清理生产缓存。
```

**Raw output**

```text
进展：目前可能是缓存的问题，尚未确认，先查看日志。  
待办：不要直接清理生产缓存。  
日志查看：先看日志，确认问题根源。
```

Repeats the logging task and adds a separate root-cause-confirmation request.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
目前可能是缓存的问题，还没有确认，先看日志，不要直接清理生产缓存。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-repair-en

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
Let us meet Wednesday, actually Thursday afternoon, at the usual place.
```

**Raw output**

```text
Let us meet Wednesday, actually Thursday afternoon, at the usual place.
```

Thursday afternoon supersedes Wednesday; usual location retained, but oral correction remains.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Let us meet Wednesday, actually Thursday afternoon, at the usual place.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-investigate-zh

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
分析 OrderList.tsx 里的 TS2345 报错，先解释原因，不要提交代码。
```

**Raw output**

```text
TS2345 是 TypeScript 中的一个常见错误，表示一个 **类型断言**，但该断言 **未被接受**。

在 `OrderList.tsx` 中，如果存在如下情况：

```tsx
const list = [1, 2, 3];
```

或者：

```tsx
const list = [1, 2, 3], [4, 5, 6]];
```

那么，TS2345 错误的原因是：

- **类型断言未被接受**。
- **类型断言未被正确使用**。

因此，TS
```

Invents a diagnosis and TypeScript code instead of rewriting the investigation request; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
分析 OrderList.tsx 里的 TS2345 报错，先解释原因，不要提交代码。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-investigate-en

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
Investigate TS2345 in OrderList.tsx, explain the cause first, and do not submit code.
```

**Raw output**

```text
TS2345: This error typically occurs when a non-nullable property is assigned a null value.

Cause: The `OrderList.tsx` file is attempting to render a component that expects a non-null value, but the actual value is null.

Do not submit code.
```

Invents a null-property diagnosis without source evidence; no-submit clause alone is insufficient.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
Investigate TS2345 in OrderList.tsx, explain the cause first, and do not submit code.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-question-zh

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
你能帮我看看这个接口为什么这么慢吗？
```

**Raw output**

```text
Sure, I can help you investigate why the interface is so slow. 

To proceed, I would need some information:

1. **What is the interface?** (e.g., a REST API, a GraphQL API, etc.)

2. **What are the endpoints and their behaviors?**

3. **What are the performance metrics?** (e.g., response time, number of requests, etc.)

4. **Are there any logs or traces that could help identify the issue?
```

Answers in English with follow-up questions rather than relaying the Chinese question; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
你能帮我看看这个接口为什么这么慢吗？
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-repair-zh

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
我们约周三，不对，改成周四下午，地点还是老地方。
```

**Raw output**

```text
我们约周三，不对，改成 **周四下午**，地点还是老地方。
```

Thursday afternoon correction and usual location retained, but stale Wednesday/correction phrasing remains.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
我们约周三，不对，改成周四下午，地点还是老地方。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-risk-en

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
This might be a cache issue but it is not confirmed, check the logs first and do not clear the production cache.
```

**Raw output**

```text
进展：此问题可能是缓存问题，但尚未确认。请先检查日志，不要清除生产缓存。
```

Translates to Chinese rather than preserving message language; tentative cause and no-clear restriction remain.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
This might be a cache issue but it is not confirmed, check the logs first and do not clear the production cache.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-review-zh

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
李老师您好，报告初稿已附上，麻烦周二之前看一下，有问题随时联系，谢谢。
```

**Raw output**

```text
李老师您好，  

报告初稿已附上，麻烦您在**周二之前**审阅并提出意见。  

如有问题，请随时联系。  

感谢您的审阅！  

祝好！
```

Adds a request to provide feedback and an invented closing, expanding the original review/contact request.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
李老师您好，报告初稿已附上，麻烦周二之前看一下，有问题随时联系，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-delay-zh

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
王先生您好，样品还没到，预计周五到货，收到后我再安排测试，谢谢。
```

**Raw output**

```text
邮件内容如下：

---

**主题：样品到货通知**  

**尊敬的王先生：**  

样品尚未到货，预计将于 **周五** 到达。  

收到样品后，我将安排 **测试**。  

感谢您的理解与配合。  

**谢谢。**  

--- 

如需进一步调整，请随时告知。
```

Adds a subject, wrapper, unsupported courtesy and an assistant offer to revise the message.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
王先生您好，样品还没到，预计周五到货，收到后我再安排测试，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-leave-en

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
Hi Rowan, I need Thursday off, I handed my work to Priya, please approve it, thanks.
```

**Raw output**

```text
Hi Rowan,

I need Thursday off. I have handed my work to Priya. Please approve it. Thanks.
```

Thursday leave, Priya handoff and approval request retained with natural salutation/body.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Rowan, I need Thursday off, I handed my work to Priya, please approve it, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-payment-en

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
The payment API is finished, the test environment is not ready, start integration when it is ready, and do not release today.
```

**Raw output**

```text
进展：支付 API 已完成开发。  
待办：测试环境尚未就绪，需等待测试环境恢复。  
集成：待测试环境恢复后，方可进行系统集成。  
发布：今日不发布。
```

Translates to Chinese and introduces environment-recovery framing not stated by the source.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
The payment API is finished, the test environment is not ready, start integration when it is ready, and do not release today.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## dev / scene / dev-uncertain-zh

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
我大概六点到，还不能确定，你们先别点我的饭。
```

**Raw output**

```text
我大概六点到，还不能确定，你们先别点我的饭。
```

Approximate six arrival, uncertainty and no-food-order request preserved.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
我大概六点到，还不能确定，你们先别点我的饭。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-reschedule-en

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
Hi Cameron, the meeting moves from Monday to Tuesday at three in the afternoon, the online link stays the same, please confirm, thanks.
```

**Raw output**

```text
Hi Cameron,

The meeting moves from Monday to Tuesday at three in the afternoon. The online link stays the same. Please confirm, thanks.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Cameron, the meeting moves from Monday to Tuesday at three in the afternoon, the online link stays the same, please confirm, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-upload-zh

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
给 /api/upload 加文件大小校验，最大十兆，保持错误码 E413，不要修改成功响应。
```

**Raw output**

```text
给 /api/upload 加文件大小校验，最大十兆，保持错误码 E413，不要修改成功响应。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
给 /api/upload 加文件大小校验，最大十兆，保持错误码 E413，不要修改成功响应。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-availability-en

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
Hi Avery, I might miss Friday.s review, it is not confirmed, and I will let you know once I am sure, sorry.
```

**Raw output**

```text
Hi Avery,

I might miss Friday's review. It's not confirmed yet, and I'll let you know once I'm sure. Sorry.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: unsupported_content

**Cleanup**

```text
Hi Avery, I might miss Friday.s review, it is not confirmed, and I will let you know once I am sure, sorry.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-condition-en

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
If it rains tomorrow, switch to online, otherwise meet downstairs as planned.
```

**Raw output**

```text
If it rains tomorrow, switch to online, otherwise meet downstairs as planned.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
If it rains tomorrow, switch to online, otherwise meet downstairs as planned.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-retry-zh

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
给 sendInvoice 加失败重试，最多三次，两次之间等十秒，不要重试权限错误。
```

**Raw output**

```text
给 sendInvoice 添加失败重试，最多三次。两次之间等十秒。不要重试权限错误。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
给 sendInvoice 加失败重试，最多三次，两次之间等十秒，不要重试权限错误。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-retry-en

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
Add retries to sendInvoice, at most three attempts with ten seconds between attempts, and do not retry permission errors.
```

**Raw output**

```text
Add retries to sendInvoice, at most three attempts with ten seconds between attempts, and do not retry permission errors.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Add retries to sendInvoice, at most three attempts with ten seconds between attempts, and do not retry permission errors.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-invoice-zh

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
陈女士您好，发票已经寄出，快递单号 AB123，请查收，有问题联系我，谢谢。
```

**Raw output**

```text
陈女士您好：

发票已经寄出，快递单号为 AB123，请查收。如有问题，请联系我，谢谢。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
陈女士您好，发票已经寄出，快递单号 AB123，请查收，有问题联系我，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-export-zh

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
导出功能完成了，安全评审还没过，等评审通过再上线，负责人是小郑。
```

**Raw output**

```text
导出功能完成了，安全评审还没过，等评审通过再上线，负责人是小郑。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
导出功能完成了，安全评审还没过，等评审通过再上线，负责人是小郑。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-upload-en

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
Add a file-size check to /api/upload with a ten MB limit, keep error E413 and do not change the success response.
```

**Raw output**

```text
Add a file-size check to /api/upload with a ten MB limit, keep error E413 and do not change the success response.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Add a file-size check to /api/upload with a ten MB limit, keep error E413 and do not change the success response.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-request-en

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
Can you check why these rewrites are discarded, without changing code yet?
```

**Raw output**

```text
Can you check why these rewrites are discarded, without changing code yet?
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Can you check why these rewrites are discarded, without changing code yet?
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-incident-zh

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
今天接口恢复了，但是数据还在补，小何负责补数，小宋核对结果，明早再通知客户。
```

**Raw output**

```text
今天接口恢复了，但是数据还在补，小何负责补数，小宋核对结果，明早再通知客户。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
今天接口恢复了，但是数据还在补，小何负责补数，小宋核对结果，明早再通知客户。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-incident-en

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
The API is back, data backfill is still running, Alex owns backfill, Blair checks results, and notify customers tomorrow morning.
```

**Raw output**

```text
The API is back, data backfill is still running, Alex owns backfill, Blair checks results, and notify customers tomorrow morning.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
The API is back, data backfill is still running, Alex owns backfill, Blair checks results, and notify customers tomorrow morning.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-estimate-zh

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
内存上涨可能跟连接池有关，暂时还不能确定，先加监控，不要重启线上服务。
```

**Raw output**

```text
内存上涨可能跟连接池有关，暂时还不能确定，先加监控，不要重启线上服务。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
内存上涨可能跟连接池有关，暂时还不能确定，先加监控，不要重启线上服务。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-correction-en

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
The budget is not eight hundred, it is nine hundred, still four people, please do not mix that up.
```

**Raw output**

```text
The budget is not eight hundred, it is nine hundred, still four people, please do not mix that up.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
The budget is not eight hundred, it is nine hundred, still four people, please do not mix that up.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-migration-zh

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
只分析 migrate.go 为什么执行失败，保留数据库现状，先不要运行迁移。
```

**Raw output**

```text
只分析 migrate.go 为什么执行失败，保留数据库现状，先不要运行迁移。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
只分析 migrate.go 为什么执行失败，保留数据库现状，先不要运行迁移。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-migration-en

Scene: prompt → chat (requested by fixture); template: live-v2

**Source**

```text
Only investigate why migrate.go fails, preserve the database state and do not run migrations yet.
```

**Raw output**

```text
Only investigate why migrate.go fails, preserve the database state, and do not run migrations yet.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
Only investigate why migrate.go fails, preserve the database state and do not run migrations yet.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-request-zh

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
你看一下为什么这些改写会被丢弃，先别改代码。
```

**Raw output**

```text
你看一下为什么这些改写会被丢弃，先别改代码。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
你看一下为什么这些改写会被丢弃，先别改代码。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-correction-zh

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
预算不是八百，是九百，人数还是四个人，别弄错了。
```

**Raw output**

```text
预算是九百，人数还是四个人，别弄错了。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
预算不是八百，是九百，人数还是四个人，别弄错了。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-estimate-en

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
The memory increase may involve the connection pool, we are not sure, add monitoring first and do not restart production.
```

**Raw output**

```text
The memory increase may involve the connection pool. We are not sure. Add monitoring first and do not restart production.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
The memory increase may involve the connection pool, we are not sure, add monitoring first and do not restart production.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-reschedule-zh

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
张先生您好，会议从周一改到周二下午三点，线上链接不变，麻烦确认，谢谢。
```

**Raw output**

```text
张先生您好：

会议时间从周一改到周二，时间安排为下午三点。线上链接不变。

麻烦您确认一下，谢谢。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: unsupported_content

**Cleanup**

```text
张先生您好，会议从周一改到周二下午三点，线上链接不变，麻烦确认，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-availability-zh

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
刘老师您好，我可能赶不上周五的评审，还没确定，确定后会尽快告知，抱歉。
```

**Raw output**

```text
刘老师您好：

我可能赶不上周五的评审，尚未确定。确定后会尽快告知，抱歉。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
刘老师您好，我可能赶不上周五的评审，还没确定，确定后会尽快告知，抱歉。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-invoice-en

Scene: email → email (requested by fixture); template: live-v2

**Source**

```text
Hi Morgan, the invoice has been sent, tracking is AB123, please check it and contact me with any issues, thanks.
```

**Raw output**

```text
Hi Morgan,

The invoice has been sent. Tracking number is AB123. Please check it and contact me with any issues.

Thanks.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Morgan, the invoice has been sent, tracking is AB123, please check it and contact me with any issues, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-export-en

Scene: work → chat (requested by fixture); template: live-v2

**Source**

```text
The export feature is complete, security review is still pending, release only after approval, and Quinn owns it.
```

**Raw output**

```text
The export feature is complete, security review is still pending, release is only after approval, and Quinn owns it.
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: False; strict: accepted

**Cleanup**

```text
The export feature is complete, security review is still pending, release only after approval, and Quinn owns it.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / production / test-condition-zh

Scene: chat → chat (requested by fixture); template: live-v2

**Source**

```text
如果明天还下雨，我们就改线上，没下雨还是在楼下见。
```

**Raw output**

```text
如果明天还下雨，我们就改线上，没下雨还是在楼下见。
```

All inspected source facts, dates, roles, restrictions and request intent retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
如果明天还下雨，我们就改线上，没下雨还是在楼下见。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-reschedule-en

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
Hi Cameron, the meeting moves from Monday to Tuesday at three in the afternoon, the online link stays the same, please confirm, thanks.
```

**Raw output**

```text
Hi Cameron,

The meeting is scheduled to move from Monday to Tuesday at 3:00 PM. The online link will remain the same.

Please confirm.

Thanks.
```

Monday-to-Tuesday move and three PM equivalence, same link and confirmation request all retained.

Faithful: True; scene ready: True; strict: new_number

**Cleanup**

```text
Hi Cameron, the meeting moves from Monday to Tuesday at three in the afternoon, the online link stays the same, please confirm, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-upload-zh

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
给 /api/upload 加文件大小校验，最大十兆，保持错误码 E413，不要修改成功响应。
```

**Raw output**

```text
Status: 文件大小校验已添加到 /api/upload 接口。  
Next step: 确保最大文件大小为 10MB。  
Error code: E413（保持原错误码）。  
Success response: 不修改成功响应内容。
```

Changes the requested file-size validation into an already-added completion claim.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
给 /api/upload 加文件大小校验，最大十兆，保持错误码 E413，不要修改成功响应。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-availability-en

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
Hi Avery, I might miss Friday.s review, it is not confirmed, and I will let you know once I am sure, sorry.
```

**Raw output**

```text
Hi Avery,  

I might miss Friday's review. It's not confirmed yet, and I'll let you know once I'm sure.  

Sorry.
```

Repairs Friday.s punctuation; tentative absence, uncertainty, future notice and apology retained.

Faithful: True; scene ready: True; strict: unsupported_content

**Cleanup**

```text
Hi Avery, I might miss Friday.s review, it is not confirmed, and I will let you know once I am sure, sorry.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-condition-en

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
If it rains tomorrow, switch to online, otherwise meet downstairs as planned.
```

**Raw output**

```text
If it rains tomorrow, switch to online, otherwise meet downstairs as planned.
```

Both rain/online and no-rain/downstairs branches retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
If it rains tomorrow, switch to online, otherwise meet downstairs as planned.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-retry-zh

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
给 sendInvoice 加失败重试，最多三次，两次之间等十秒，不要重试权限错误。
```

**Raw output**

```text
```text
Status: sendInvoice 加失败重试机制，最多三次重试。
Requirements:
- 两次重试之间等待 10 秒。
- 不重试权限错误。
Action: 开始执行 sendInvoice 操作，同时启动失败重试机制，最多三次重试。
```
```

Adds an instruction to start executing sendInvoice and invented procedural output.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
给 sendInvoice 加失败重试，最多三次，两次之间等十秒，不要重试权限错误。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-retry-en

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
Add retries to sendInvoice, at most three attempts with ten seconds between attempts, and do not retry permission errors.
```

**Raw output**

```text
**Revised Request:**

Add retries to the `sendInvoice` function. The retry mechanism should be as follows:

- **Maximum number of attempts:** 3  
- **Delay between attempts:** 10 seconds  
- **Do not retry on permission errors.**

---

**Implementation Notes:**

- The `sendInvoice` function should be modified to include a retry mechanism.  
- The retry logic should be implemented using a loop that runs up to 3 times, with a 10-second delay between attempts.  
- The retry should be disabled if the error is a permission error.
```

Adds implementation notes and an explicit loop requirement, beyond rewriting the request.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
Add retries to sendInvoice, at most three attempts with ten seconds between attempts, and do not retry permission errors.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-invoice-zh

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
陈女士您好，发票已经寄出，快递单号 AB123，请查收，有问题联系我，谢谢。
```

**Raw output**

```text
陈女士您好，  

发票已寄出，快递单号为 AB123，请查收。  

如有问题，请联系我。  

谢谢！
```

Sent invoice, AB123 tracking, check/contact requests and thanks retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
陈女士您好，发票已经寄出，快递单号 AB123，请查收，有问题联系我，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-export-zh

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
导出功能完成了，安全评审还没过，等评审通过再上线，负责人是小郑。
```

**Raw output**

```text
进展：导出功能已完成。  
待办：安全评审尚未通过，需等待评审结果。  
上线计划：待安全评审通过后，方可上线。  
负责人：小郑。
```

Completed feature, pending security review, approval-before-release and Zheng ownership clearly separated.

Faithful: True; scene ready: True; strict: length

**Cleanup**

```text
导出功能完成了，安全评审还没过，等评审通过再上线，负责人是小郑。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-upload-en

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
Add a file-size check to /api/upload with a ten MB limit, keep error E413 and do not change the success response.
```

**Raw output**

```text
Status: Added file-size check to `/api/upload` with a 10 MB limit.

Next step: Error E413 is retained for existing error handling.

Success response remains unchanged as per original requirements.

No changes to the public API.

No implementation of a fix.

File-size check is added with a 10 MB limit.
```

Changes the request into completed work and adds API/no-fix assertions.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
Add a file-size check to /api/upload with a ten MB limit, keep error E413 and do not change the success response.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-request-en

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
Can you check why these rewrites are discarded, without changing code yet?
```

**Raw output**

```text
I will check why these rewrites are discarded, without changing code yet.
```

Changes the recipient-directed question into a speaker commitment to investigate.

Faithful: False; scene ready: False; strict: accepted

**Cleanup**

```text
Can you check why these rewrites are discarded, without changing code yet?
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-incident-zh

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
今天接口恢复了，但是数据还在补，小何负责补数，小宋核对结果，明早再通知客户。
```

**Raw output**

```text
进展：接口已恢复，数据仍在补数中。  
待办：小何负责数据补数，小宋负责核对结果。  
下一步：明早再通知客户，数据补数工作仍在进行中。
```

API restoration, ongoing backfill, He/Song responsibilities and tomorrow-morning notice retained in clear sections.

Faithful: True; scene ready: True; strict: length

**Cleanup**

```text
今天接口恢复了，但是数据还在补，小何负责补数，小宋核对结果，明早再通知客户。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-incident-en

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
The API is back, data backfill is still running, Alex owns backfill, Blair checks results, and notify customers tomorrow morning.
```

**Raw output**

```text
进展：API 已恢复，数据回填仍在运行。  
当前状态：  
- **Alex** 负责回填。  
- **Blair** 检查结果。  
- **通知客户**：明天上午通知客户。
```

Translates to Chinese, despite keeping the owner/action relationships.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
The API is back, data backfill is still running, Alex owns backfill, Blair checks results, and notify customers tomorrow morning.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-estimate-zh

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
内存上涨可能跟连接池有关，暂时还不能确定，先加监控，不要重启线上服务。
```

**Raw output**

```text
进展：内存上涨可能与连接池有关，目前尚无法确定原因。  
待办：增加内存监控，避免重启线上服务。
```

Uncertain connection-pool cause, monitor-first and no-restart intent retained in concise sections.

Faithful: True; scene ready: True; strict: unsupported_content

**Cleanup**

```text
内存上涨可能跟连接池有关，暂时还不能确定，先加监控，不要重启线上服务。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-correction-en

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
The budget is not eight hundred, it is nine hundred, still four people, please do not mix that up.
```

**Raw output**

```text
The budget is not eight hundred, it is nine hundred, still four people, please do not mix that up.
```

Nine-hundred final budget, rejection of eight hundred and four people retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
The budget is not eight hundred, it is nine hundred, still four people, please do not mix that up.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-migration-zh

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
只分析 migrate.go 为什么执行失败，保留数据库现状，先不要运行迁移。
```

**Raw output**

```text
分析 `migrate.go` 为什么执行失败，保留数据库现状，不运行迁移。

---

### **问题分析**

1. **执行失败的原因**  
   - `migrate.go` 中的迁移逻辑可能有错误，例如：
     - 错误的 SQL 语句。
     - 错误的表结构定义。
     - 错误的字段类型或约束。
   - `migrate.go` 中的迁移逻辑可能没有正确处理表结构或字段约束。
   - `migrate.go` 中的迁移逻辑
```

Adds speculative SQL/schema explanations instead of rewriting the limited investigation request; truncated.

Faithful: False; scene ready: False; strict: truncated

**Cleanup**

```text
只分析 migrate.go 为什么执行失败，保留数据库现状，先不要运行迁移。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-migration-en

Scene: prompt → prompt (requested by fixture); template: scene-alignment-v1

**Source**

```text
Only investigate why migrate.go fails, preserve the database state and do not run migrations yet.
```

**Raw output**

```text
Investigation: Why `migrate.go` fails.

**Requirements:**

- Preserve the database state before any migrations.
- Do not run migrations yet.
- Investigate why `migrate.go` fails, without implementing a fix.

**Next Steps:**

- Review the behavior of `migrate.go`.
- Check for any errors or exceptions that are being thrown.
- Investigate why the migration is failing, without implementing a fix.
```

Adds procedural next steps and reframes state preservation as before migrations rather than preserving current state.

Faithful: False; scene ready: False; strict: length

**Cleanup**

```text
Only investigate why migrate.go fails, preserve the database state and do not run migrations yet.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-request-zh

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
你看一下为什么这些改写会被丢弃，先别改代码。
```

**Raw output**

```text
我看到为什么这些改写会被丢弃，先别改代码。
```

Changes a request for the recipient to inspect into a statement by the speaker that they have seen the cause.

Faithful: False; scene ready: False; strict: accepted

**Cleanup**

```text
你看一下为什么这些改写会被丢弃，先别改代码。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-correction-zh

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
预算不是八百，是九百，人数还是四个人，别弄错了。
```

**Raw output**

```text
预算是九百，人数还是四个人，别弄错了。
```

Resolves correction to nine hundred and preserves four people/no-mix-up request.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
预算不是八百，是九百，人数还是四个人，别弄错了。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-estimate-en

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
The memory increase may involve the connection pool, we are not sure, add monitoring first and do not restart production.
```

**Raw output**

```text
进展：内存增加可能涉及连接池，目前尚不确定，因此先添加监控，不重启生产。
```

Translates to Chinese while preserving tentative cause and no-restart instruction.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
The memory increase may involve the connection pool, we are not sure, add monitoring first and do not restart production.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-reschedule-zh

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
张先生您好，会议从周一改到周二下午三点，线上链接不变，麻烦确认，谢谢。
```

**Raw output**

```text
张先生您好，

会议时间从周一改到周二，时间安排为下午三点。线上链接保持不变。

请确认上述安排，谢谢。
```

Monday-to-Tuesday three-PM move, unchanged link and confirmation request retained.

Faithful: True; scene ready: True; strict: unsupported_content

**Cleanup**

```text
张先生您好，会议从周一改到周二下午三点，线上链接不变，麻烦确认，谢谢。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-availability-zh

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
刘老师您好，我可能赶不上周五的评审，还没确定，确定后会尽快告知，抱歉。
```

**Raw output**

```text
刘老师您好，  

我可能赶不上周五的评审，尚未确定。  

确定后我会尽快告知，敬请理解。  

谢谢。
```

Drops the spoken apology, replacing it with requests for understanding and thanks.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
刘老师您好，我可能赶不上周五的评审，还没确定，确定后会尽快告知，抱歉。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-invoice-en

Scene: email → email (requested by fixture); template: scene-alignment-v1

**Source**

```text
Hi Morgan, the invoice has been sent, tracking is AB123, please check it and contact me with any issues, thanks.
```

**Raw output**

```text
Hi Morgan,

The invoice has been sent. Tracking number is AB123.

Please check it and contact me with any issues.

Thanks.
```

Sent status, AB123, check/contact requests and thanks retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
Hi Morgan, the invoice has been sent, tracking is AB123, please check it and contact me with any issues, thanks.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-export-en

Scene: work → work (requested by fixture); template: scene-alignment-v1

**Source**

```text
The export feature is complete, security review is still pending, release only after approval, and Quinn owns it.
```

**Raw output**

```text
进展：导出功能已完成。  
待办：安全审查仍需进行。  
发布要求：仅在获得批准后才能发布。  
所有权：Quinn拥有该资源。
```

Translates the message to Chinese, with awkward ownership wording.

Faithful: False; scene ready: False; strict: unsupported_content

**Cleanup**

```text
The export feature is complete, security review is still pending, release only after approval, and Quinn owns it.
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

## test / scene / test-condition-zh

Scene: chat → chat (requested by fixture); template: scene-alignment-v1

**Source**

```text
如果明天还下雨，我们就改线上，没下雨还是在楼下见。
```

**Raw output**

```text
如果明天还下雨，我们就改线上，没下雨还是在楼下见。
```

Both rain/online and no-rain/downstairs branches retained.

Faithful: True; scene ready: True; strict: accepted

**Cleanup**

```text
如果明天还下雨，我们就改线上，没下雨还是在楼下见。
```

Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.

