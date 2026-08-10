import os
import json

# 从环境变量中获取 __workflow__ 的值
workflow_env = os.getenv('__workflow__')
metadata_env = os.getenv('__metadata__')
user_env = os.getenv('__user__')
# 解析 JSON 数据
workflow_data = json.loads(workflow_env)
metadata_data = json.loads(metadata_env)
user_data = json.loads(user_env)
print(workflow_data['workflow'])
print(metadata_data)
print(user_data['device'])