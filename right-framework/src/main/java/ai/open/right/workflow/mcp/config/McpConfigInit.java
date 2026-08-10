package ai.open.right.workflow.mcp.config;

import java.util.Map;

// 初始化MCP配置
public interface McpConfigInit {

    public void init(Map<String, Object> config) throws Exception;
}
