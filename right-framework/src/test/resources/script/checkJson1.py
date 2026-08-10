import json

def validate_json(json_str):
    try:
        # 解析 JSON 字符串为 Python 字典
        data = json.loads(json_str)
        # 检查 'mobile' 键是否存在于字典中
        if 'mobile' not in data:
            raise ValueError("JSON 中缺少 mobile 属性")
        # 将 'mobile' 的值转换为字符串，以便检查长度
        mobile_str = str(data['mobile'])
        # 检查手机号码长度是否为 10 位
        if len(mobile_str) != 10:
            raise ValueError("错误的手机号码")
        # 如果检查通过，返回原始的 JSON 字符串
        return json_str
    except json.JSONDecodeError:
        # 若 JSON 解析失败，抛出异常
        raise ValueError("输入的不是有效的 JSON 字符串")


# 示例 JSON 字符串
json_str = '{"mobile":1234567890,"value":"你好"}'
try:
    result = validate_json(json_str)
    print(result)
except ValueError as e:
    print(e)