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
    # 清理文件名中的非法字符
    safe_name = task_info.replace("[", "").replace("]", "").replace("-", "_").replace(" ", "_").replace("#", "N")
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
        # 设置超时时间，stream 模式下 timeout 指的是连接超时
        response = requests.post(url=url, headers=headers, json=request_data, timeout=600, stream=is_stream)

        # 1. 检查 HTTP 状态码
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
                    # 2. 检查业务 code
                    biz_code = data_json.get("code")
                    if biz_code is not None and biz_code != 200:
                        reason = f"流式业务Code异常: {biz_code}"
                        save_failed_log(task_info, url, headers, request_data, json_str, 200, reason)
                        print(f"❌ {task_info} 失败 ({reason})")
                        return False
                except json.JSONDecodeError:
                    continue

            print(f"✅ {task_info} 成功")
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
                reason = f"解析JSON失败: {str(e)}"
                save_failed_log(task_info, url, headers, request_data, resp_text, 200, reason)
                print(f"❌ {task_info} 失败 ({reason})")
                return False

            print(f"✅ {task_info} 成功")
            return True

    except Exception as e:
        all_content = "\n".join(raw_responses) if raw_responses else str(e)
        save_failed_log(task_info, url, headers, request_data, all_content, "ERR", f"异常: {str(e)}")
        print(f"❌ {task_info} 异常")
        return False

if __name__ == "__main__":
    BASE_URL = "http://127.0.0.1:9998"

    # --- 并发配置 ---
    CONCURRENT_THREADS_PER_MODE = 5  # 每个模式 5 个线程
    TOTAL_REQUESTS_PER_THREAD = 100   # 每个线程跑 100 次 (总计 5*100=500 次/模式)
    PROVIDER_NAME = "vertex"

    # 总计 10 个线程 (5 Stream + 5 Non-Stream)
    max_total_workers = CONCURRENT_THREADS_PER_MODE * 2

    print(f"🚀 压力测试启动")
    print(f"🚀 并发配置：5路 Stream 并发 + 5路 Non-Stream 并发 (总计 10 线程)")
    print(f"🚀 判定标准：HTTP 200 且 业务 JSON code == 200")
    print("-" * 80)

    total_success = 0
    total_planned = CONCURRENT_THREADS_PER_MODE * 2 * TOTAL_REQUESTS_PER_THREAD

    with ThreadPoolExecutor(max_workers=max_total_workers) as executor:
        futures = []

        # 启动 5 个 Non-Stream 并发分支
        for thread_id in range(1, CONCURRENT_THREADS_PER_MODE + 1):
            ns_headers = {"Authorization": "quick_start@cr", "Content-Type": "application/json"}
            ns_data = {
                "model": "",
                "messages": [{"role": "user", "content": COMMON_CONTENT}],
                "metadata": {"__provider": PROVIDER_NAME},
                "stream": False
            }
            # 每个线程内部循环执行
            for loop_id in range(1, TOTAL_REQUESTS_PER_THREAD + 1):
                task_label = f"[{PROVIDER_NAME}-NonStream-T{thread_id}-#{loop_id}]"
                futures.append(executor.submit(execute_request, BASE_URL, ns_headers, ns_data, task_label))

        # 启动 5 个 Stream 并发分支
        for thread_id in range(1, CONCURRENT_THREADS_PER_MODE + 1):
            s_headers = {"Authorization": "quick_start@cr_stream", "Content-Type": "application/json"}
            s_data = {
                "model": "",
                "messages": [{"role": "user", "content": COMMON_CONTENT}],
                "metadata": {"__provider": PROVIDER_NAME},
                "stream": True
            }
            for loop_id in range(1, TOTAL_REQUESTS_PER_THREAD + 1):
                task_label = f"[{PROVIDER_NAME}-Stream-T{thread_id}-#{loop_id}]"
                futures.append(executor.submit(execute_request, BASE_URL, s_headers, s_data, task_label))

        # 汇总结果
        for future in as_completed(futures):
            if future.result():
                total_success += 1

    print("-" * 80)
    print(f"🏁 测试结束 | 总成功数: {total_success}/{total_planned}")
    print(f"📊 最终成功率: {(total_success/total_planned)*100:.2f}%")
    print(f"📂 错误详情请查看: {FAILED_DIR}")