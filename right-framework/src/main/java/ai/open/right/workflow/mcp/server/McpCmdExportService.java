package ai.open.right.workflow.mcp.server;

import ai.open.right.workflow.flow.config.McpExportConfig;

// MCP实际指令处理
public interface McpCmdExportService {

    // 发布配置
    public void export(McpExportConfig mcpExportConfig) throws Exception;

    // 处理指令
    public void cmd(McpRequest mcpRequest) throws Exception;
}
