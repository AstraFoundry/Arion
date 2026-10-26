-- rime.lua
-- rime-llm-reranker · v6 中英文混合记录版

-- ─── luasocket 路径注入（Apple Silicon Mac）─────────────
local _home = os.getenv('HOME') or ''

package.path = package.path
  ..';'.._home..'/.luarocks/share/lua/5.4/?.lua'
  ..';'.._home..'/.luarocks/share/lua/5.4/?/init.lua'
  ..';/opt/homebrew/share/lua/5.4/?.lua'
  ..';/opt/homebrew/share/lua/5.4/?/init.lua'

package.cpath = package.cpath
  ..';'.._home..'/.luarocks/lib/lua/5.4/?.so'
  ..';/opt/homebrew/lib/lua/5.4/?.so'

-- ─── luasocket 检测 ──────────────────────────────────────
local _socket_ok, _socket_http, _socket_ltn12
do
  local ok1, m1 = pcall(require, 'socket.http')
  local ok2, m2 = pcall(require, 'ltn12')
  if ok1 and ok2 then
    _socket_ok   = true
    _socket_http = m1
    _socket_ltn12 = m2
  end
end

-- ─── JSON 编码 ───────────────────────────────────────────
local function json_encode(v)
  local ESC = {['"']='\\"',['\\']='\\\\', ['\n']='\\n',['\r']='\\r',['\t']='\\t'}
  local function esc(s)
    s = s:gsub('["\\\n\r\t]', ESC)
    s = s:gsub('[\x00-\x1f]', function(c) return string.format('\\u%04x',c:byte()) end)
    return s
  end
  local function is_arr(t)
    local n=0
    for k in pairs(t) do
      if type(k)~='number' or k~=math.floor(k) or k<1 then return false end
      n=n+1
    end
    return n==#t
  end
  local function enc(x, d)
    d=d or 0
    local t=type(x)
    if     t=='nil'     then return 'null'
    elseif t=='boolean' then return x and 'true' or 'false'
    elseif t=='number'  then
      if x~=x or math.abs(x)==math.huge then return 'null' end
      if x==math.floor(x) and math.abs(x)<1e15 then return string.format('%d',x) end
      return string.format('%.17g',x)
    elseif t=='string' then return '"'..esc(x)..'"'
    elseif t=='table'  then
      if is_arr(x) then
        local p={} for i=1,#x do p[i]=enc(x[i],d+1) end
        return '['..table.concat(p,',')..']'
      else
        local p={}
        for k,val in pairs(x) do
          if type(k)=='string' then
            table.insert(p, '"'..esc(k)..'":'..enc(val,d+1))
          end
        end
        return '{'..table.concat(p,',')..'}'
      end
    end
    return 'null'
  end
  local ok, r = pcall(enc, v)
  return ok and r or nil
end

local function json_extract_content(s)
  if not s or s=='' then return nil end
  local c = s:match('"content"%s*:%s*"(.-[^\\])"')
  if not c then return nil end
  c = c:gsub('\\"','"'):gsub('\\n','\n'):gsub('\\t','\t'):gsub('\\\\','\\')
  c = c:match('^%s*(.-)%s*$') or c
  if c:find('%?') or c:find('\xef\xbf\xbd') then return nil end
  return c ~= '' and c or nil
end

-- ─── HTTP POST ───────────────────────────────────────────
local function http_post(url, payload_table, timeout)
  local body = json_encode(payload_table)
  if not body then return nil, 'encode failed' end

  if _socket_ok then
    _socket_http.TIMEOUT = timeout
    local parts = {}
    local _, code = _socket_http.request({
      url    = url,
      method = 'POST',
      headers = {
        ['content-type']   = 'application/json',
        ['content-length'] = tostring(#body),
        ['authorization']  = 'Bearer sk-placeholder',
      },
      source = _socket_ltn12.source.string(body),
      sink   = _socket_ltn12.sink.table(parts),
    })
    if code == 200 then return table.concat(parts) end
    return nil, 'luasocket HTTP '..tostring(code)
  else
    local safe = body:gsub("'","'\"'\"'")
    local cmd = string.format(
      "curl -s --connect-timeout %.2f --max-time %.2f "..
      "-X POST -H 'Content-Type: application/json' "..
      "-H 'Authorization: Bearer sk-placeholder' "..
      "-d '%s' '%s' 2>/dev/null",
      timeout*0.5, timeout, safe, url)
    local h = io.popen(cmd,'r')
    if not h then return nil,'popen failed' end
    local r = h:read('*all'); h:close()
    if r and #r>0 then return r end
    return nil,'curl empty response'
  end
end

-- ─── 配置读取 ────────────────────────────────────────────
local function get_config(env)
  local c = env.engine.schema.config
  local function gb(k,d) local v=c:get_bool(k);   return v~=nil and v or d end
  local function gs(k,d) return c:get_string(k) or d end
  local function gn(k,d) return c:get_double(k) or d end
  local function gi(k,d) return c:get_int(k)    or d end
  local max_cands = gi('llm_reranker/max_candidates', 5)
  return {
    enabled      = gb('llm_reranker/enabled', true),
    endpoint     = gs('llm_reranker/endpoint', 'http://localhost:8000/v1/chat/completions'),
    model        = gs('llm_reranker/model',    'Qwen3.5-0.8B-MLX-bf16'),
    max_cands    = max_cands,
    timeout      = gn('llm_reranker/timeout', 0.5),
    ctx_len      = gi('llm_reranker/context_length', 60),
    temperature  = gn('llm_reranker/temperature', 0.0),
    max_tokens   = gi('llm_reranker/max_tokens', 20),
    min_syllables   = gi('llm_reranker/min_syllables', 1),
    min_ctx_len     = gi('llm_reranker/min_context_len', 0),
    max_preedit_len = gi('llm_reranker/max_preedit_len', 10),
    max_rerank_pos  = gi('llm_reranker/max_rerank_pos', 5),
    system_prompt   = gs('llm_reranker/system_prompt',
      '你是输入法候选词排序助手。'..
      '根据对话历史和当前输入，从候选词列表中选出最符合语境的词。'..
      '按优先级从高到低列出最多5个候选词，用顿号分隔，不要任何解释。'),
    fallback = gb('llm_reranker/fallback_on_error', true),
    debug    = gb('llm_reranker/debug_log', false),
    collect_train = gb('llm_reranker/collect_training_data', true),
    collect_user_pairs = gb('llm_reranker/collect_user_pairs', true),
    collect_llm_pairs = gb('llm_reranker/collect_llm_pairs', false),
    max_pairwise_negatives = gi('llm_reranker/max_pairwise_negatives', math.max(max_cands - 1, 1)),
  }
end

-- ─── 模块级配置 ──────────────────────────────────────────
local _cfg = {}

-- ─── 调试日志 ─────────────────────────────────────────────
local LOG_PATH   = _home..'/Library/Rime/rime_llm.log'

local function dlog(msg)
  if not _cfg.debug then return end
  local f = io.open(LOG_PATH,'a')
  if f then f:write(os.date('%H:%M:%S')..' '..msg..'\n'); f:close() end
end

-- ─── 训练数据日志 ─────────────────────────────────────────
local TRAIN_LOG_PATH = _home..'/Library/Rime/rime_train.jsonl'

-- UTF-8 安全截断
local function utf8_tail(s, max_bytes)
  if #s <= max_bytes then return s end
  local start = #s - max_bytes + 1
  while start <= #s and s:byte(start) >= 0x80 and s:byte(start) <= 0xBF do
    start = start + 1
  end
  return s:sub(start)
end

-- 保留候选文本首次出现的位置，避免重复候选污染 pairwise 标签。
local function unique_texts(texts)
  local result = {}
  local seen = {}
  for _, text in ipairs(texts or {}) do
    if text and text ~= '' and not seen[text] then
      table.insert(result, text)
      seen[text] = true
    end
  end
  return result
end

-- Rime 分段选词时 preedit 可能是“已选文本 + 剩余编码”，例如“有没you”。
local function parse_preedit(preedit)
  if not preedit or preedit == '' then return nil end

  local inline_text, code = preedit:match("^(.-)([A-Za-z' ]+)$")
  if not code or code == '' then return nil end

  return {
    raw = preedit,
    inline_text = inline_text or '',
    code = code,
  }
end

local function log_train_pair(context, pinyin, positive, negative, source, meta)
  if not _cfg.collect_train then return end
  if not positive or not negative or positive == negative then return end
  if positive == '' or negative == '' then return end

  -- Pairwise 训练记录以同一上下文和拼音下的“好候选 > 差候选”为最小样本。
  local payload = {
    context  = context,
    pinyin   = pinyin,
    positive = positive,
    negative = negative,
    source   = source,
  }
  if meta then
    for k, v in pairs(meta) do payload[k] = v end
  end

  local record = json_encode(payload)
  if not record then return end

  local f = io.open(TRAIN_LOG_PATH, 'a')
  if f then
    f:write(record..'\n')
    f:close()
  end
end

-- 记录 LLM 排序的相邻 pair；默认关闭，仅作为弱监督伪标签使用。
local function log_llm_ranking_pairs(context, pinyin, ranked_texts)
  if not _cfg.collect_train then return end
  if not _cfg.collect_llm_pairs then return end
  if not ranked_texts or #ranked_texts < 2 then return end

  -- 纯简拼过滤
  if pinyin:lower():match('^[bcdfghjklmnpqrstxyz]+$') then
    dlog('TRAIN: skip jianpin '..pinyin)
    return
  end
  -- 可选：无上下文时跳过（若希望保留则注释）
  -- if not context or #context < _cfg.min_ctx_len then
  --   dlog('TRAIN: skip empty/short context')
  --   return
  -- end

  for i = 1, #ranked_texts - 1 do
    log_train_pair(context, pinyin, ranked_texts[i], ranked_texts[i+1], 'llm_rank', {
      positive_pos = i,
      negative_pos = i + 1,
      label_quality = 'pseudo',
    })
  end
end

-- 查找词语在有序候选文本列表中的位置。
local function index_of(list, word)
  for i, w in ipairs(list) do
    if w == word then return i end
  end
  return nil
end

-- 分段选词时 commit_history 可能给完整文本，训练匹配需要还原成候选词本身。
local function selected_candidate_text(snap, committed)
  if not snap or not committed then return committed end
  local inline_text = snap.inline_text or ''
  if inline_text ~= '' and committed:sub(1, #inline_text) == inline_text then
    local suffix = committed:sub(#inline_text + 1)
    if suffix ~= '' then return suffix end
  end
  return committed
end

-- 从用户最终上屏行为生成 pairwise 样本，保证正负例来自同一次候选展示。
local function log_user_selection_pairs(snap, selected)
  if not _cfg.collect_train or not _cfg.collect_user_pairs then return end
  if not snap or not snap.ordered_texts or #snap.ordered_texts < 2 then return end
  if not selected or selected == '' then return end

  local shown_candidates = unique_texts(snap.ordered_texts)
  if #shown_candidates < 2 then return end

  local selected_pos = index_of(shown_candidates, selected)
  if not selected_pos then return end

  local source = selected_pos == 1 and 'user_accept' or 'user_select'
  local logged = 0
  for pos, negative in ipairs(shown_candidates) do
    local is_comparable = false
    if selected_pos == 1 then
      is_comparable = pos > 1
    else
      -- 手动选中靠后的候选时，只把排在它前面的展示候选视为明确负例。
      is_comparable = pos < selected_pos
    end

    if is_comparable and negative ~= selected then
      log_train_pair(snap.context, snap.pinyin, selected, negative, source, {
        positive_pos = selected_pos,
        negative_pos = pos,
        shown_candidates = shown_candidates,
        label_quality = 'user',
      })
      logged = logged + 1
      if logged >= _cfg.max_pairwise_negatives then break end
    end
  end

  if logged > 0 then
    dlog(string.format('USER_PAIR: %s pos=%d pairs=%d pinyin=%s',
      selected, selected_pos, logged, snap.pinyin))
  end
end

-- ─── 音节数估算 ──────────────────────────────────────────
local INITIALS = {
  'zh','ch','sh',
  'b','p','m','f',
  'd','t','n','l',
  'g','k','h',
  'j','q','x',
  'z','c','s',
  'r','y','w',
}

local function count_syllables(preedit)
  if not preedit or preedit == '' then return 0 end
  local s = preedit:lower()
  local count = 0
  local i = 1
  while i <= #s do
    local matched = false
    for _, ini in ipairs(INITIALS) do
      if s:sub(i, i+#ini-1) == ini then
        count = count + 1
        i = i + #ini
        while i <= #s and s:sub(i,i):match('[aeiouüv]') do i = i + 1 end
        if s:sub(i,i+1) == 'ng' then i = i + 2
        elseif s:sub(i,i) == 'n' then i = i + 1 end
        matched = true
        break
      end
    end
    if not matched then
      if s:sub(i,i):match('[aeiouüv]') then
        count = count + 1
        while i <= #s and s:sub(i,i):match('[aeiouüvng]') do i = i + 1 end
      else
        i = i + 1
      end
    end
  end
  return count
end

-- ─── 候选词匹配（仅精确匹配） ──────────────────────────────
local function find_match(list, word)
  if not word or word == '' then return nil end
  word = word:match('^%s*(.-)%s*$') or word
  for i, c in ipairs(list) do
    if c.text == word then return i end
  end
  return nil
end

-- ─── 解析 LLM 响应 ───────────────────────────────────────
local function extract_ranking(response, list)
  local content = json_extract_content(response)
  if not content then return nil, {} end

  local ranked_texts = {}
  local seen = {}
  for raw_word in content:gmatch('[^、,，;；\n]+') do
    local word = raw_word:match('^%s*(.-)%s*$') or raw_word
    word = word:gsub('^[%d①②③%.、。%s]+', '')
    word = word:match('^%s*(.-)%s*$') or word
    if word ~= '' and not seen[word] then
      local idx = find_match(list, word)
      if idx then
        table.insert(ranked_texts, list[idx].text)
        seen[word] = true
      else
        break  -- 无法精确匹配，停止解析
      end
    end
  end

  local best_idx = nil
  if #ranked_texts > 0 then
    for i, c in ipairs(list) do
      if c.text == ranked_texts[1] then best_idx = i; break end
    end
  end

  return best_idx, ranked_texts
end

-- ─── LLM 调用 ────────────────────────────────────────────
local function query_llm(cfg, committed, preedit, cand_texts)
  local user_content
  if committed ~= '' then
    user_content =
      '对话历史：「'..committed..'」\n'..
      '当前输入拼音：「'..preedit..'」\n'..
      '候选词：'..table.concat(cand_texts,'、')..'\n'..
      '按最符合对话语境的顺序列出前5个候选词（顿号分隔）：'
  else
    user_content =
      '输入拼音：「'..preedit..'」\n'..
      '候选词：'..table.concat(cand_texts,'、')..'\n'..
      '按最可能的顺序列出前5个候选词（顿号分隔）：'
  end

  local payload = {
    model       = cfg.model,
    stream      = false,
    temperature = cfg.temperature,
    max_tokens  = cfg.max_tokens,
    messages    = {
      {role='system', content=cfg.system_prompt},
      {role='user',   content=user_content},
    },
  }

  local t0 = os.clock()
  local resp, err = http_post(cfg.endpoint, payload, cfg.timeout)
  local ms = (os.clock()-t0)*1000

  if not resp then
    dlog(string.format('FAIL %.0fms %s', ms, tostring(err)))
    return nil, ms
  end

  local content = json_extract_content(resp)
  dlog(string.format('OK %.0fms → [%s] preedit=「%s」 ctx=「%s」',
    ms, content or 'nil', preedit, utf8_tail(committed, 15)))

  return resp, ms
end

-- ─── 重排序 ──────────────────────────────────────────────
local function reorder(list, idx)
  if not idx or idx == 1 then return list end
  local r = {list[idx]}
  for i,c in ipairs(list) do if i ~= idx then table.insert(r,c) end end
  return r
end

-- ─── 模块级状态 ──────────────────────────────────────────
local committed_buffer = ''
local _last_commit_text = ''      -- 去重用
local _logged          = false

local _cache      = {}
local _cache_size = 0
local CACHE_MAX   = 300

local _querying = {}

-- 训练快照：上一次展示给用户的候选列表
local _last_snapshot = nil

-- ─── 拼音完整性判断 ──────────────────────────────────────

local function is_jianpin(preedit)
  return preedit:lower():match('^[bcdfghjklmnpqrstxyz]+$') ~= nil
end

local COMPLETE_ENDINGS = {
  'ang','eng','ing','ong','ung',
  'ian','uan','uen','van',
  'an','en','in','un','vn',
  'ng','er',
}
local function preedit_ends_incomplete(preedit)
  if not preedit or preedit == '' then return true end
  local s = preedit:lower()
  if is_jianpin(s) then return false end
  if s:sub(-1):match('[aeiouüv]') then return false end
  for _, ending in ipairs(COMPLETE_ENDINGS) do
    if #s >= #ending and s:sub(-#ending) == ending then return false end
  end
  return true
end

-- ─── 主动收集上下文（代替回调） ──────────────────────────
local function update_context_from_history(env)
  local latest = nil
  pcall(function()
    local hist = env.engine.context.commit_history
    if hist and not hist:empty() then
      local entry = hist:back()
      if entry and entry.text then
        latest = entry.text
      end
    end
  end)

  if latest and latest ~= '' and latest ~= _last_commit_text then
    local snap = _last_snapshot
    local commit_text = latest
    if snap and snap.inline_text and snap.inline_text ~= '' then
      if latest:sub(1, #snap.inline_text) ~= snap.inline_text then
        commit_text = snap.inline_text .. latest
      end
    end

    -- 发生了新的上屏；分段选词时补齐已选 inline 片段。
    committed_buffer = utf8_tail(committed_buffer .. commit_text, 1200)
    _last_commit_text = latest

    dlog('COMMIT: '..commit_text..' | buf_len='..#committed_buffer)

    -- 记录用户真实选择产生的 pairwise 偏好（利用最近一次候选快照）
    log_user_selection_pairs(snap, selected_candidate_text(snap, latest))

    -- 清空缓存，允许对新的上下文重新查询 LLM
    _cache = {}
    _cache_size = 0
    _last_snapshot = nil
  end
end

-- ─── Rime filter 入口 ────────────────────────────────────
function llm_reranker(input, env)
  local all = {}
  for cand in input:iter() do table.insert(all, cand) end

  _cfg = get_config(env)   -- 更新模块配置

  if not _logged and _cfg.debug then
    dlog('v6 started, backend: '..(_socket_ok and 'luasocket' or 'curl'))
    _logged = true
  end

  -- 主动收集上下文（必须在处理候选词之前）
  update_context_from_history(env)

  if not _cfg.enabled or #all < 2 then
    for _,c in ipairs(all) do yield(c) end
    return
  end

  local preedit = ''
  pcall(function()
    local p = env.engine.context:get_preedit()
    if p then preedit = p.text or '' end
  end)

  local parsed_preedit = parse_preedit(preedit)
  if not parsed_preedit then
    dlog('SKIP: unsupported preedit '..preedit)
    _last_snapshot = nil
    for _,c in ipairs(all) do yield(c) end
    return
  end

  local query_preedit = parsed_preedit.code
  local inline_text = parsed_preedit.inline_text
  local context_base = utf8_tail(committed_buffer .. inline_text, 1200)
  local preedit_is_english = (query_preedit:match('^[%a]+$') ~= nil)

  local syllables = count_syllables(query_preedit)
  -- 英文输入跳过音节数检查
  if syllables < _cfg.min_syllables and not preedit_is_english then
    for _,c in ipairs(all) do yield(c) end
    return
  end

  local skip = false
  local ctx_bytes = context_base:len()
  if ctx_bytes < _cfg.min_ctx_len then
    skip = true
    dlog('SKIP: no context ('..ctx_bytes..' bytes < '.._cfg.min_ctx_len..')')
  end

  local preedit_len = query_preedit:len()
  if preedit_len > _cfg.max_preedit_len then
    skip = true
    dlog('SKIP: preedit too long ('..preedit_len..' > '.._cfg.max_preedit_len..')')
  end

  local top, rest = {}, {}
  for i,c in ipairs(all) do
    if i <= _cfg.max_cands then table.insert(top,c)
    else                       table.insert(rest,c) end
  end

  local idx = nil

  if not skip then
    local history_key = utf8_tail(context_base, _cfg.ctx_len)
    -- 缓存键统一小写（英文不区分大小写）
    local cache_key = (history_key..':'..query_preedit):lower()
    local cached      = _cache[cache_key]

    if cached == nil and not _querying[cache_key] then
      _querying[cache_key] = true

      local texts = {}
      for _,c in ipairs(top) do table.insert(texts, c.text) end

      local resp = nil
      local ctx_snapshot = utf8_tail(context_base, _cfg.ctx_len)
      if _cfg.fallback then
        pcall(function()
          resp = (query_llm(_cfg, ctx_snapshot, query_preedit, texts))
        end)
      else
        resp = (query_llm(_cfg, ctx_snapshot, query_preedit, texts))
      end

      local raw_idx, ranked_texts = extract_ranking(resp or '', top)

      -- 训练数据记录：英文或拼音完整时允许记录
      local allow_rank_log = preedit_is_english or
                             (not is_jianpin(query_preedit) and not preedit_ends_incomplete(query_preedit))
      if allow_rank_log then
        log_llm_ranking_pairs(ctx_snapshot, query_preedit, ranked_texts)
      end

      if raw_idx and raw_idx > _cfg.max_rerank_pos then
        dlog(string.format('FILTERED: rank %d > max %d, skip rerank',
          raw_idx, _cfg.max_rerank_pos))
        raw_idx = nil
      end

      local ordered_texts = {}
      if raw_idx then
        for _,c in ipairs(reorder(top, raw_idx)) do table.insert(ordered_texts, c.text) end
      else
        for _,c in ipairs(top) do table.insert(ordered_texts, c.text) end
      end

      cached = {
        idx = raw_idx or false,
        texts = ordered_texts,
      }
      _querying[cache_key] = nil
      if _cache_size >= CACHE_MAX then _cache={}; _cache_size=0 end
      _cache[cache_key] = cached
      _cache_size = _cache_size + 1

      if _cfg.debug then
        if cached.idx and cached.idx > 1 then
          dlog('RERANKED: '..top[1].text..' → '..top[cached.idx].text)
        else
          dlog('NO_CHANGE: kept '..top[1].text)
        end
      end
    elseif _querying[cache_key] then
      dlog('skip: query in progress '..cache_key)
    else
      dlog('cache hit ['..tostring(cached and cached.idx)..'] '..cache_key)
    end

    if cached and cached.idx then
      idx = cached.idx
    end
  end

  local ordered = reorder(top, idx)

  -- 保存快照供下次 user_select 使用
  if _cfg.collect_train then
    local ordered_texts = {}
    for _, c in ipairs(ordered) do table.insert(ordered_texts, c.text) end
    _last_snapshot = {
      context       = utf8_tail(context_base, _cfg.ctx_len),
      pinyin        = query_preedit,
      inline_text   = inline_text,
      ordered_texts = ordered_texts,
    }
  end

  for _,c in ipairs(ordered) do yield(c) end
  for _,c in ipairs(rest)    do yield(c) end
end

-- ─── init / fini ─────────────────────────────────────────
function llm_reranker_init(env)
  _cfg = get_config(env)
  committed_buffer  = ''
  _last_commit_text = ''
  _logged           = false
  _cache            = {}
  _cache_size       = 0
  _querying         = {}
  _last_snapshot    = nil

  dlog('INIT done (v6)')
end

function llm_reranker_fini(env)
  committed_buffer  = ''
  _last_commit_text = ''
  _cache            = {}
  _cache_size       = 0
  _last_snapshot    = nil
end

-- 废弃的按键处理器（保留空函数）
function llm_context_tracker(key_event, env)
  return 2  -- kNoop
end
