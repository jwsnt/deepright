#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# seedream-gen.sh — Seedream (豆包) 图片生成 API
# 支持 文生图 / 单图生图 / 多图生图
# 凭据通过 DeepRight integration token 动态获取
# ============================================================

VERSION="1.1.1"
INTEGRATION="/Applications/DeepRight.app/Contents/MacOS/integration"
TOOL_NAME="$(basename "$0")"

# --------------- 默认参数 ---------------
PROMPT=""
SIZE="1920x1920"
N="1"
RESPONSE_FORMAT="url"
OUTPUT_FILE=""
TIMEOUT="120"
QUIET=false
IMAGES=()

# --------------- 帮助 ---------------
usage() {
    cat << 'EOF'
Usage: seedream-gen.sh --prompt PROMPT [OPTIONS]

通过 Seedream 模型生成图片，凭据从 integration token --provider seedream 动态获取。

  模式:
    文生图:           仅 --prompt
    图生图(单图):     --prompt + --image URL
    图生图(多图):     --prompt + --image URL1 --image URL2 ...

Options:
  --prompt TEXT         生成提示词（必填）
  --image URL           参考图片 URL（可多次指定，用于图生图/多图融合）
  --size WxH           图片尺寸（默认: 1920x1920，最小 3686400 像素）
  --n N                生成数量（默认: 1）
  --format FORMAT      返回格式: url | b64_json（默认: url）
  --output FILE        将结果 JSON 写入文件（默认输出到 stdout）
  --timeout SECONDS    API 超时秒数（默认: 120）
  --quiet              静默模式，仅输出图片 URL
  --help               显示此帮助
  --version            显示版本

Examples:
  # 文生图
  seedream-gen.sh --prompt "a red apple on white background"

  # 单图生图
  seedream-gen.sh --prompt "turn this into watercolor style" --image "https://example.com/photo.jpg"

  # 多图融合
  seedream-gen.sh --prompt "merge into one scene" --image "https://a.com/1.jpg" --image "https://b.com/2.jpg"

  # 自定义尺寸 + 静默输出
  seedream-gen.sh --prompt "cyberpunk city" --size 1920x1080 --quiet
EOF
    exit 0
}

version() {
    echo "$TOOL_NAME v$VERSION"
    exit 0
}

# --------------- 解析参数 ---------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --prompt)    PROMPT="$2"; shift 2 ;;
        --image)     IMAGES+=("$2"); shift 2 ;;
        --size)      SIZE="$2"; shift 2 ;;
        --n)         N="$2"; shift 2 ;;
        --format)    RESPONSE_FORMAT="$2"; shift 2 ;;
        --output)    OUTPUT_FILE="$2"; shift 2 ;;
        --timeout)   TIMEOUT="$2"; shift 2 ;;
        --quiet)     QUIET=true; shift ;;
        --help)      usage ;;
        --version)   version ;;
        *)           echo "[ERROR] Unknown option: $1" >&2; usage ;;
    esac
done

# --------------- 必填检查 ---------------
if [[ -z "$PROMPT" ]]; then
    echo "[ERROR] --prompt is required" >&2
    exit 1
fi

# --------------- 获取凭据 ---------------
if [[ ! -x "$INTEGRATION" ]]; then
    echo "[ERROR] integration binary not found or not executable: $INTEGRATION" >&2
    exit 2
fi

TOKEN_JSON="$("$INTEGRATION" token --provider seedream 2>&1)" || {
    echo "[ERROR] failed to fetch token from integration" >&2
    exit 2
}

parse_json_field() {
    local field="$1"
    echo "$TOKEN_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin).get('seedream',{}); print(d.get('$field',''))" 2>/dev/null
}

API_KEY="$(parse_json_field 'token')"
API_URL="$(parse_json_field '__url')"
MODEL="$(parse_json_field '__model_multi_output')"

if [[ -z "$API_KEY" ]]; then
    echo "[ERROR] seedream token not configured" >&2
    exit 2
fi

if [[ -z "$API_URL" ]]; then
    echo "[ERROR] seedream URL not available" >&2
    exit 2
fi

if [[ -z "$MODEL" ]]; then
    echo "[ERROR] seedream model not configured in integration token" >&2
    exit 6
fi

$QUIET || echo "[INFO] endpoint: $API_URL" >&2
$QUIET || echo "[INFO] model: $MODEL" >&2
$QUIET || echo "[INFO] key: ${API_KEY:0:12}..." >&2
if [[ ${#IMAGES[@]} -gt 0 ]]; then
    $QUIET || echo "[INFO] mode: image-to-image (${#IMAGES[@]} reference image(s))" >&2
else
    $QUIET || echo "[INFO] mode: text-to-image" >&2
fi

# --------------- 构造请求 ---------------
# 安全处理空数组: set -u 下用 ${IMAGES[@]+"${IMAGES[@]}"} 检测
REQUEST_JSON=$(python3 -c "
import json, sys

req = {
    'model': '$MODEL',
    'prompt': '''${PROMPT//\'/\'\\\'\'}''',
    'size': '$SIZE',
    'n': $N,
    'response_format': '$RESPONSE_FORMAT'
}
images_raw = '''${IMAGES[*]:-}'''.strip()
if images_raw:
    req['images'] = images_raw.split()

print(json.dumps(req, ensure_ascii=False))
")

$QUIET || echo "[INFO] sending request..." >&2

# --------------- 调用 API ---------------
BODY_FILE=$(mktemp)

HTTP_CODE=$(curl -s -w "%{http_code}" \
    -X POST "$API_URL" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "$REQUEST_JSON" \
    --max-time "$TIMEOUT" \
    -o "$BODY_FILE" 2>/dev/null) || {
    echo "[ERROR] network error or timeout" >&2
    rm -f "$BODY_FILE"
    exit 3
}

BODY=$(cat "$BODY_FILE")
rm -f "$BODY_FILE"

# --------------- 处理响应 ---------------
if [[ "$HTTP_CODE" -ge 200 && "$HTTP_CODE" -lt 300 ]]; then
    # 检测空数据
    HAS_DATA=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('data',[])))" 2>/dev/null || echo "0")
    if [[ "$HAS_DATA" == "0" ]]; then
        echo "[ERROR] generation succeeded but returned no image data" >&2
        echo "$BODY" >&2
        exit 5
    fi

    if [[ -n "$OUTPUT_FILE" ]]; then
        echo "$BODY" > "$OUTPUT_FILE"
        $QUIET || echo "[OK] HTTP $HTTP_CODE, result saved to $OUTPUT_FILE" >&2
    elif $QUIET; then
        echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for item in d.get('data', []):
    if 'url' in item:
        print(item['url'])
    elif 'b64_json' in item:
        print(item['b64_json'])
" 2>/dev/null
    else
        echo "$BODY"
    fi
else
    echo "[ERROR] API returned HTTP $HTTP_CODE" >&2
    echo "$BODY" >&2
    exit 4
fi
