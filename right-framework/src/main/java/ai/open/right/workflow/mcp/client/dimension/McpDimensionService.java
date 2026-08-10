package ai.open.right.workflow.mcp.client.dimension;

import ai.open.right.workflow.flow.WorkflowTask;

public interface McpDimensionService {

    public McpDimension buildDimension(McpDimension dimension, WorkflowTask workTask) throws Exception;
}
