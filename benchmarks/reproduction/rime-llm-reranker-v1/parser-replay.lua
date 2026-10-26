-- Replay real recorded responses through the original parser; no inference.
local json = require('dkjson')
dofile(arg[1])
local parser
for i = 1, 100 do
  local name, value = debug.getupvalue(llm_reranker, i)
  if not name then break end
  if name == 'extract_ranking' then parser = value; break end
end
assert(parser, 'original parser upvalue missing')
for line in io.lines() do
  local request = assert(json.decode(line))
  local list = {}
  for i, text in ipairs(request.candidates) do list[i] = {text = text} end
  local best, ranked = parser(request.response, list)
  print(json.encode({id = request.id, best = best or false, ranked = ranked}))
end
