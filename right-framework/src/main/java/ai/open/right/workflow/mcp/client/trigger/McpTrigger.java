package ai.open.right.workflow.mcp.client.trigger;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;

import java.util.Map;

public interface McpTrigger {

    public static final String NAME = "globalMcpTrigger";

    public void beforeToolsCall(McpDimension mcpDimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception;

    public void beforePromptGet(McpDimension mcpDimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception;

    public void beforeResourcesRead(McpDimension mcpDimension, String uri, WorkflowTask workTask) throws Exception;
}
