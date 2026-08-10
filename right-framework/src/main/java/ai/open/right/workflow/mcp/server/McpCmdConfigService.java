package ai.open.right.workflow.mcp.server;

import ai.open.right.workflow.flow.config.McpExportConfig;

// 注册Export配置
public interface McpCmdConfigService {

    public static final String NAME_RESOURCES_TEMPLATES = "CONFIG_RESOURCES_TEMPLATES";

    public static final String NAME_RESOURCES = "CONFIG_RESOURCES";

    public static final String NAME_PROMPTS = "CONFIG_PROMPTS";

    public static final String NAME_TOOLS = "CONFIG_TOOLS";

    public void export(String name, McpExportConfig mcpExportConfig) throws Exception;

    public McpExportConfig fetch(String name) throws Exception;
}
