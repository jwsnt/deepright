package ai.open.right.workflow.mcp.client.dimension;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.llm.store.Dimension;
import lombok.Builder;
import lombok.Getter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;

import java.util.Map;

@Getter
@Builder
@ToString
// MCP维度
public class McpDimension implements Dimension {

    // Header（可通过自定义Assistant注入）
    protected Map<String, String> headers;

    // MCP配置
    protected McpConfig mcpConfig;

    protected String workflow;

    protected String device;

    // MCP Client
    protected String client;

    // MCP Name
    protected String name;

    protected String chat;

    protected String biz;

    public McpDimension merge(FunctionContext functionContext) {
        this.merge(functionContext.getWorkTask());
        return this;
    }

    public McpDimension merge(WorkflowTask workTask) {
        this.workflow = StringUtils.defaultIfBlank(this.workflow, workTask.getWorkflow());
        this.device = StringUtils.defaultIfBlank(this.device, workTask.getDevice());
        this.chat = StringUtils.defaultIfBlank(this.chat, workTask.getChat());
        this.biz = StringUtils.defaultIfBlank(this.biz, workTask.getBiz());
        return this;
    }

    public McpDimension bind(String[] pair) {
        this.client = pair[0];
        this.name = pair[1];
        return this;
    }

    @Override
    // 维度字符串
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }

    // 是否配置了McpListener（监听）
    public Boolean hasListener() {
        return this.mcpConfig != null && this.mcpConfig.hasRewriter();
    }

    // 是否配置了McpTrigger（触发）
    public Boolean hasTrigger() {
        return this.mcpConfig != null && this.mcpConfig.hasTrigger();
    }
}
