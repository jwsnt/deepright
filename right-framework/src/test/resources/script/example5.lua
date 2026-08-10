-- 引入 dkjson 库用于解析 JSON
local json = require('dkjson')

-- 解析 __workflow__ 变量中的 JSON 字符串
local workflowData = json.decode(__workflow__)
local metadata = json.decode(__metadata__)
local user = json.decode(__user__)

-- 检查解析是否成功
if workflowData then
    -- 输出解析后的数据
    print('Workflow: ' .. workflowData.workflow)
    print('Biz: ' .. workflowData.biz)
else
    -- 解析失败时输出错误信息
    print('JSON 解析失败')
end

-- 检查解析是否成功
if metadata then
    -- 输出解析后的数据
    print('Metadata: ' .. metadata.HELLO)
else
    -- 解析失败时输出错误信息
    print('JSON 解析失败')
end

-- 检查解析是否成功
if user then
    -- 输出解析后的数据
    print('User: ' .. user.device)
else
    -- 解析失败时输出错误信息
    print('JSON 解析失败')
end