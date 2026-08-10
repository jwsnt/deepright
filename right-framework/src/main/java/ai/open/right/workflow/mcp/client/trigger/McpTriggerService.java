package ai.open.right.workflow.mcp.client.trigger;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;

import java.util.Map;

// MCP触发监听（请求前）
public interface McpTriggerService {

    public void beforeToolsCall(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception;

    public void beforePromptGet(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception;

    public void beforeResourcesRead(McpDimension dimension, String uri, WorkflowTask workTask) throws Exception;
}
