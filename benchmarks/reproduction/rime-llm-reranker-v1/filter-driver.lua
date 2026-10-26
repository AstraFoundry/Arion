-- Adapt recorded Rime candidates to the unchanged upstream filter API.
-- Every HTTP request uses the real local server. No evaluator labels are read.
local json = require('dkjson')
local clock = require('benchclock')
local http = require('socket.http')
local original_request = http.request
local requests = {}
http.request = function(options)
  local body, response = {}, {}
  local source, sink = options.source, options.sink
  options.source = function()
    local chunk, err = source()
    if chunk then table.insert(body, chunk) end
    return chunk, err
  end
  options.sink = function(chunk, err)
    if chunk then table.insert(response, chunk) end
    return sink(chunk, err)
  end
  local start = clock.milliseconds()
  local ok, code, headers, status = original_request(options)
  table.insert(requests, {request = table.concat(body), response = table.concat(response),
    networkWallMs = clock.milliseconds() - start, code = code, status = status})
  return ok, code, headers, status
end

dofile(arg[1])
for line in io.lines() do
  local input, _, err = json.decode(line)
  assert(input and not err, err)
  for key in pairs(input) do
    assert(key == 'id' or key == 'config' or key == 'history' or key == 'steps', 'unexpected request field')
  end
  local config = input.config
  local function get(_, key) return config[key:gsub('^llm_reranker/', '')] end
  local schema = {config = {get_bool = get, get_string = get, get_int = get, get_double = get}}
  local current, history_text = nil, input.history or ''
  local history = {empty = function() return history_text == '' end,
    back = function() return {text = history_text} end}
  local env = {engine = {schema = schema, context = {commit_history = history,
    get_preedit = function() return {text = current.preedit} end}}}
  llm_reranker_init(env)
  local output = {}
  for _, step in ipairs(input.steps) do
    current = step
    if step.history ~= nil then history_text = step.history end
    local candidates = {}
    for i, text in ipairs(step.candidates) do candidates[i] = {text = text, id = i} end
    local stream = {iter = function() local i = 0; return function() i = i + 1; return candidates[i] end end}
    local yielded, identities = {}, {}
    _G.yield = function(candidate)
      table.insert(yielded, candidate.text)
      table.insert(identities, candidate.id)
    end
    requests = {}
    local start, cpu_start = clock.milliseconds(), os.clock()
    local ok, failure = pcall(llm_reranker, stream, env)
    table.insert(output, {id = step.id, preedit = step.preedit, source = step.candidates,
      ranked = yielded, identities = identities, wallMs = clock.milliseconds() - start,
      processCpuMs = (os.clock() - cpu_start) * 1000, requests = requests,
      status = ok and 'completed' or 'failed', failure = failure})
  end
  llm_reranker_fini(env)
  print(json.encode({id = input.id, steps = output}))
  io.stdout:flush()
end
