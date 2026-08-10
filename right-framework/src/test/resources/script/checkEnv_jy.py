import os
import json
# 解析 JSON 数据
workflow_data = json.loads(__workflow__)
metadata_data = json.loads(__metadata__)
user_data = json.loads(__user__)
print(workflow_data['workflow'])
print(metadata_data)
print(user_data['device'])