package ai.open.right.workflow.flow.assistant.mcp;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;

@Setter
@Getter
@Slf4j
public class McpAssistant extends DefaultAssistant {

    protected McpDimensionService mcpDimensionService;

    protected McpClientService mcpClientService;

    // 构建MCP维度
    protected McpDimension buildMcpDimension(String workflow, McpConfig mcpConfig, WorkflowTask workTask) throws Exception {
        McpDimension mcpDimension = McpDimension.builder()
                .device(workTask.getUserContext().getDevice())
                .chat(workTask.getChat())
                .biz(workTask.getBiz())
                .mcpConfig(mcpConfig)
                .workflow(workflow)
                .build();
        if (log.isDebugEnabled()) {
            log.debug("McpDimension={}", mcpDimension);
        }
        return this.buildMcpDimension(mcpDimension, workTask);
    }

    protected McpDimension buildMcpDimension(McpDimension mcpDimension, WorkflowTask workTask) throws Exception {
        return this.mcpDimensionService.buildDimension(mcpDimension, workTask);
    }

    // 构建MCP Runtime（运行时）
    protected McpRuntime buildMcpRuntime(McpConfig mcpConfig, WorkflowTask workTask) throws Exception {
        McpRuntime mcpRuntime = McpRuntime.builder()
                .timeout(mcpConfig.getTimeout())
                .dynamic(mcpConfig.getDynamic())
                .prefix(mcpConfig.getPrefix())
                .suffix(mcpConfig.getSuffix())
                .workTask(workTask)
                .build();
        if (log.isDebugEnabled()) {
            log.debug("McpRuntime={}", mcpRuntime);
        }
        return mcpRuntime;
    }

    @Setter
    @Getter
    public static class McpInitConfig extends DefInitConfig {

        @Autowired
        protected McpDimensionService mcpDimensionService;

        @Autowired
        protected McpClientService mcpClientService;
    }
}