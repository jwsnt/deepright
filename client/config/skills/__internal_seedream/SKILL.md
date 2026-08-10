---
name: __internal_seedream
description: "文生图、单图/多图生图。凭据自动获取，支持自定义尺寸/数量/返回格式"
---

## 快速使用
```bash
./seedream-gen.sh --prompt "主题描述"                         # 文生图
./seedream-gen.sh --prompt "描述" --image "https://..."       # 单图参考
./seedream-gen.sh --prompt "描述" --image "A" --image "B"     # 多图融合
./seedream-gen.sh --prompt "" --size 1920x1080 --n 2 --quiet  # 自定义
./seedream-gen.sh --prompt "" --output result.json            # 存文件
```

## 参数
```
--prompt         必填,文字描述
--image          可选,参考图URL(可重复,公网可访问HTTP/HTTPS,不支持base64)
--size           可选,默认1920x1920(最小3,686,400像素)
--n              可选,默认1(生成张数)
--quiet          可选,仅输出图片URL
--output         可选,保存JSON到文件
--timeout        可选,默认120s
```

## 退出码
| 码 | 含义 |
|---|---|
| 0 | 成功 |
| 1 | 缺--prompt |
| 2 | token获取失败 |
| 3 | 网络超时/不可达 |
| 4 | API非2xx |
| 5 | 返回空数据
| 6 | integration未配model字段 |

## 前置
- 依赖: bash3.2+, curl7.0+, python3.6+
- 凭据自动从 `integration token --provider seedream` 获取(需DeepRight.app)
- token字段→API_KEY, `__url`→API_URL, `__model_multi_output`→MODEL
- 返回URL有效期24h

## 限制
- 只HTTP/HTTPS URL参考图,不支持base64内联
- prompt建议<2000字符
- 不含视频/Inpaint/局部编辑
