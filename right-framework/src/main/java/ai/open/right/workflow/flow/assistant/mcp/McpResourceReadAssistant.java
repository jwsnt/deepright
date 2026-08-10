package ai.open.right.workflow.flow.assistant.mcp;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
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

import java.util.Map;

@Slf4j
@Setter
@Getter
// Mcp Resource Read协议的Assistant
public class McpResourceReadAssistant extends McpAssistant {

    public static final String WORKFLOW_NAME = "def-mcpResourceRead";

    // 用于二次查找配置
    protected WorkflowConfigService workflowConfigService;

    // MCP响应重写（请求后）
    protected McpRewriteService mcpRewriteService;

    // MCP触发监听（请求前）
    protected McpTriggerService mcpTriggerService;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        McpConfig mcpConfig = workflowConfig.getMcpConfig();
        McpResult<String> result = null;
        if (mcpConfig != null) {
            McpDimension mcpDimension = this.buildMcpDimension(workTask.getWorkflow(), mcpConfig, workTask);
            McpRuntime mcpRuntime = this.buildMcpRuntime(mcpConfig, workTask);
            // MCP触发监听（请求前）
            this.mcpTriggerService.beforeResourcesRead(mcpDimension, mcpConfig.getName(), workTask);
            result = super.mcpClientService.resourcesRead(mcpDimension.getClient(), mcpDimension.getName(), mcpRuntime, mcpDimension);
            // MCP响应重写（请求后）
            result = this.mcpRewriteService.resourcesRead(mcpDimension, mcpConfig.getName(), workTask, result);
        } else {
            // 使用上游（Upstream）获取McpConfig并创建McpDimension
            mcpConfig = this.workflowConfigService.config(workTask, workTask.getUpstream()).getMcpConfig();
            McpDimension mcpDimension = this.buildMcpDimension(workTask.getUpstream(), mcpConfig, workTask);
            String uri = this.fetchUri(workTask);
            // MCP触发监听（请求前）
            this.mcpTriggerService.beforeResourcesRead(mcpDimension, uri, workTask);
            // FetchUri从MCP Response里提取URI再次请求
            result = super.mcpClientService.resourcesRead(workTask.getWorkflow(), uri, mcpDimension);
            // MCP响应重写（请求后）
            result = this.mcpRewriteService.resourcesRead(mcpDimension, uri, workTask, result);
        }
        Assert.notNull(result.getResult(), "Mcp resource result can not be empty: " + result);
        this.chainOr2Endpoint(workflowConfig, workTask, result.getResult());
    }

    // 从MCP Resource Read获取URI
    protected String fetchUri(WorkflowTask workTask) throws Exception {
        Map<String, String> params = workTask.getObjectQuery(Map.class);
        Assert.notEmpty(params, "MCP resource response cannot be empty");
        String uri = params.get("uri");
        Assert.hasText(uri, "MCP resource URI cannot be empty: " + uri);
        return uri;
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

        @Bean(McpResourceReadAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = McpResourceReadAssistant.WORKFLOW_NAME)
        public McpResourceReadAssistant mcpResourceReadAssistant() throws Exception {
            McpResourceReadAssistant mcpResourceReadAssistant = new McpResourceReadAssistant();
            BeanUtils.copyProperties(this, mcpResourceReadAssistant);
            log.info("McpResourceReadAssistant inited");
            return mcpResourceReadAssistant;
        }
    }
}
