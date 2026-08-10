package ai.open.right.workflow.mcp.client;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.mcp.client.utils.McpContentUtils;
import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class McpBodyServiceTest {

    @Test
    public void test() throws Exception {
        Map<String, Object> content = new HashMap<String, Object>();
        content.put("text", "HELLO");
        McpContentUtils mcpBodyService = new McpContentUtils();
        Assert.assertEquals("HELLO", mcpBodyService.resource("text", content));
    }

    @Test
    public void testEmpty() throws Exception {
        Map<String, Object> content = new HashMap<String, Object>();
        content.put("text_", "HELLO");
        McpContentUtils mcpBodyService = new McpContentUtils();
        Assert.assertNull(mcpBodyService.resource("text_", content));
    }

    @Test
    public void testMulti() throws Exception {
        Map<String, Object> resources = JsonUtils.read(IOUtils.toString(ResourceUtils.getURL("classpath:MCP_Prompt_list_response.json").openStream(), StandardCharsets.UTF_8), Map.class);
        McpContentUtils mcpBodyService = new McpContentUtils();
        List<String> response = new ArrayList<String>();
        for (Object message : List.class.cast(resources.get("messages"))) {
            Map<String, Object> each = (Map<String, Object>)((Map<String, Object>) message).get("content");
            response.add(mcpBodyService.resource(String.class.cast(each.get("type")), each));
        }
        Assert.assertEquals("分析这些系统日志和代码文件是否存在问题:", response.get(0));
        Assert.assertEquals("[2024-03-14 15:32:11] 错误: network.py:127 中的连接超时\n[2024-03-14 15:32:15] 警告: 重试连接(尝试 2/3)\n[2024-03-14 15:32:20] 错误: 超过最大重试次数", response.get(1));
        Assert.assertEquals("def connect_to_service(timeout=30):\n    retries = 3\n    for attempt in range(retries):\n        try:\n            return establish_connection(timeout)\n        except TimeoutError:\n            if attempt == retries - 1:\n                raise\n            time.sleep(5)\n\ndef establish_connection(timeout):\n    # 连接实现\n    pass", response.get(2));
    }
}
