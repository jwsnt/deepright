package ai.open.right.workflow.mcp.config;

import java.util.Map;

// 加载MCP Config
public interface McpConfigService {

    public Map<String, Object> config(String uri) throws Exception;
}
