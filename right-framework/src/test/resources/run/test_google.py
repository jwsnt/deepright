import requests
import json
import os
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

# 公共请求内容
COMMON_CONTENT = """package ai.open.right;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
import org.springframework.context.annotation.PropertySource;

@PropertySource("classpath:application.properties")
@SpringBootApplication(exclude = {DataSourceAutoConfiguration.class})
public class MainApplication {
    public static void main(String[] args) throws Exception {
        SpringApplication.run(MainApplication.class, args);
    }
}"""

# 路径配置
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
FAILED_DIR = os.path.join(SCRIPT_DIR, "failed")
if not os.path.exists(FAILED_DIR):
    os.makedirs(FAILED_DIR)

def save_failed_log(task_info, url, headers, data, response_content, status_code, reason):
    """记录失败请求详情"""
    timestamp = int(time.time() * 1000)
    safe_name = task_info.replace("[", "").replace("]", "").replace("-", "_")
    file_path = os.path.join(FAILED_DIR, f"{safe_name}_{timestamp}.json")

    detail = {
        "task": task_info,
        "reason": reason,
        "status_code": status_code,
        "payload": data,
        "content_received": response_content
    }
    with open(file_path, "w", encoding="utf-8") as f:
        json.dump(detail, f, ensure_ascii=False, indent=4)
    return file_path

def execute_request(url, headers, request_data, task_info):
    """
    执行单个请求：
    1. 必须 HTTP Status == 200
    2. 报文 JSON 里的 code 属性必须 == 200
    """
    is_stream = request_data.get("stream", False)
    raw_responses = []

    try:
        response = requests.post(url=url, headers=headers, json=request_data, timeout=600, stream=is_stream)

        # 检查 HTTP 状态码
        if response.status_code != 200:
            error_msg = f"HTTP状态码错误: {response.status_code}"
            error_body = response.text if not is_stream else "Stream error body"
            save_failed_log(task_info, url, headers, request_data, error_body, response.status_code, error_msg)
            print(f"❌ {task_info} 失败 ({error_msg})")
            return False

        if is_stream:
            # 流式处理
            for line in response.iter_lines():
                if not line:
                    continue

                line_str = line.decode('utf-8').strip()
                # 兼容 SSE 格式，去掉前面的 "data: "
                json_str = line_str[5:] if line_str.startswith("data: ") else line_str

                if json_str == "[DONE]":
                    break

                try:
                    raw_responses.append(json_str)
                    data_json = json.loads(json_str)
                    # 检查业务 code
                    biz_code = data_json.get("code")
                    if biz_code is not None and biz_code != 200:
                        reason = f"流式报文业务Code异常: {biz_code}"
                        save_failed_log(task_info, url, headers, request_data, json_str, 200, reason)
                        print(f"❌ {task_info} 失败 ({reason})")
                        return False
                except json.JSONDecodeError:
                    continue # 忽略非 JSON 行

            print(f"✅ {task_info} 成功 (Stream 200)")
            return True
        else:
            # 非流式处理
            resp_text = response.text
            try:
                data_json = response.json()
                biz_code = data_json.get("code")
                if biz_code != 200:
                    reason = f"业务Code异常: {biz_code}"
                    save_failed_log(task_info, url, headers, request_data, resp_text, 200, reason)
                    print(f"❌ {task_info} 失败 ({reason})")
                    return False
            except Exception as e:
                reason = f"非流式响应解析JSON失败: {str(e)}"
                save_failed_log(task_info, url, headers, request_data, resp_text, 200, reason)
                print(f"❌ {task_info} 失败 ({reason})")
                return False

            print(f"✅ {task_info} 成功 (Non-Stream 200)")
            return True

    except Exception as e:
        all_content = "\n".join(raw_responses) if raw_responses else str(e)
        save_failed_log(task_info, url, headers, request_data, all_content, "ERR", f"异常: {str(e)}")
        print(f"❌ {task_info} 异常")
        return False

def run_task_loop(url, headers, data, task_label, count):
    """执行特定模式的循环任务"""
    success = 0
    for i in range(1, count + 1):
        full_label = f"{task_label}-{i}"
        if execute_request(url, headers, data, full_label):
            success += 1
    return success

if __name__ == "__main__":
    BASE_URL = "http://127.0.0.1:9998"
    PROVIDERS = ["gemini"]
    LOOP_COUNT = 5  # 每种模式跑5次

    total_workers = len(PROVIDERS) * 2

    print(f"🚀 开始测试：{len(PROVIDERS)} 个 Provider 并行")
    print(f"🚀 模式：每个 Provider 同时启动 Stream 和 Non-Stream 并发")
    print(f"🚀 标准：HTTP 200 且 JSON 业务 code == 200")
    print("-" * 80)

    total_success = 0
    with ThreadPoolExecutor(max_workers=total_workers) as executor:
        future_to_task = {}

        for p in PROVIDERS:
            # Non-Stream
            ns_headers = {"Authorization": "quick_start@cr", "Content-Type": "application/json"}
            ns_data = {
                "model": "",
                "messages": [{"role": "user", "content": COMMON_CONTENT}],
                "metadata": {"__provider": p},
                "stream": False
            }
            f_ns = executor.submit(run_task_loop, BASE_URL, ns_headers, ns_data, f"[{p}-NonStream]", LOOP_COUNT)
            future_to_task[f_ns] = f"{p} Non-Stream"

            # Stream
            s_headers = {"Authorization": "quick_start@cr_stream", "Content-Type": "application/json"}
            s_data = {
                "model": "",
                "messages": [{"role": "user", "content": COMMON_CONTENT}],
                "metadata": {"__provider": p},
                "stream": True
            }
            f_s = executor.submit(run_task_loop, BASE_URL, s_headers, s_data, f"[{p}-Stream]", LOOP_COUNT)
            future_to_task[f_s] = f"{p} Stream"

        for future in as_completed(future_to_task):
            task_name = future_to_task[future]
            try:
                success_count = future.result()
                total_success += success_count
                print(f"📊 任务 [{task_name}] 完成，成功率: {success_count}/{LOOP_COUNT}")
            except Exception as e:
                print(f"❗ 任务 [{task_name}] 运行异常: {e}")

    print("-" * 80)
    print(f"🏁 测试结束 | 总成功数: {total_success}/{len(PROVIDERS) * LOOP_COUNT * 2}")
    print(f"📂 失败详情查阅: {FAILED_DIR}")