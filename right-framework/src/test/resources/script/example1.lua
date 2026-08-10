```lua
-- 定义数据数组
local numbers = {1, 2, 3, 4, 5, 6, 999}
local sum = 0
-- 遍历数组并累加元素值
for _, num in ipairs(numbers) do
    sum = sum + num
end
-- 计算平均数
local average = sum / #numbers
print("这些数的平均数是: ", average)
```