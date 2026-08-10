package ai.open.right.workflow.mcp.server;

import java.util.Map;

// MCP请求
public interface McpRequest {

    // 报文内容
    public Map<String, Object> getContent();

    // Header请求
    public Map<String, String> getHeaders();

    public String getMethod();

    public String getTrace();

    public Object getId();

    // 回写，Wrap表示是否要包装为MCP Result
    public void write(McpResponse response) throws Exception;

    // 回写错误
    public void error(String message) throws Exception;
}
