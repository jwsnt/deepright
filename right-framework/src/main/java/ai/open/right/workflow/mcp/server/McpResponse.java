package ai.open.right.workflow.mcp.server;

import java.util.Map;

public interface McpResponse {

    public Map<String, Object> getResult();

    // 是否为通知型协议
    public Boolean getNotifier();

    // 是否包装Result协议
    public Boolean getWrap();
}
