package ai.open.right.workflow.flow.assistant.mcp;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.rewrtier.McpRewriteService;
import ai.open.right.workflow.mcp.client.trigger.McpTriggerService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
// MCP Prompt Get协议的Assistant
public class McpPromptGetAssistant extends McpAssistant {

    public static final String WORKFLOW_NAME = "def-mcpPromptGet";

    // 用于二次查找配置
    protected WorkflowConfigService workflowConfigService;

    // MCP响应重写（请求后）
    protected McpRewriteService mcpRewriteService;

    // MCP触发监听（请求前）
    protected McpTriggerService mcpTriggerService;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Map<String, Object> mcpContent = workTask.getObjectQuery(Map.class);
        McpConfig mcpConfig = workflowConfig.getMcpConfig();
        McpResult<List<History>> result = null;
        if (mcpConfig != null) {
            McpDimension mcpDimension = this.buildMcpDimension(workTask.getWorkflow(), mcpConfig, workTask);
            McpRuntime mcpRuntime = this.buildMcpRuntime(mcpConfig, workTask);
            // MCP触发监听（请求前）
            this.mcpTriggerService.beforePromptGet(mcpDimension, mcpContent, workTask);
            result = this.mcpClientService.promptGet(mcpDimension.getClient(), mcpDimension.getName(), mcpContent, mcpRuntime, mcpDimension);
            // MCP响应重写（请求后）
            result = this.mcpRewriteService.promptGet(mcpDimension, mcpContent, workTask, result);
        } else {
            // 使用上游（Upstream）获取McpConfig并创建McpDimension
            mcpConfig = this.workflowConfigService.config(workTask, workTask.getUpstream()).getMcpConfig();
            McpDimension mcpDimension = this.buildMcpDimension(workTask.getUpstream(), mcpConfig, workTask);
            // MCP触发监听（请求前）
            this.mcpTriggerService.beforePromptGet(mcpDimension, mcpContent, workTask);
            result = this.mcpClientService.promptGet(mcpDimension.getClient(), mcpDimension.getName(), mcpContent, mcpDimension);
            // MCP响应重写（请求后）
            result = this.mcpRewriteService.promptGet(mcpDimension, mcpContent, workTask, result);
        }
        Assert.notEmpty(result.getResult(), "Mcp prompt result can not be empty: " + workTask.getQuery() + ", " + mcpConfig);
        this.chainOr2Endpoint(workflowConfig, workTask, JsonUtils.write(result.getResult()));
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends McpInitConfig {

        @Autowired
        protected WorkflowConfigService workflowConfigService;

        @Autowired
        protected McpRewriteService mcpRewriteService;

        @Autowired
        protected McpTriggerService mcpTriggerService;

        @Bean(McpPromptGetAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = McpPromptGetAssistant.WORKFLOW_NAME)
        public McpPromptGetAssistant mcpPromptGetAssistant() throws Exception {
            McpPromptGetAssistant mcpPromptGetAssistant = new McpPromptGetAssistant();
            BeanUtils.copyProperties(this, mcpPromptGetAssistant);
            log.info("McpPromptGetAssistant inited");
            return mcpPromptGetAssistant;
        }
    }
}
