package ai.open.right.workflow.mcp.client;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;

@Setter
@Getter
public class McpContext<T> {

    protected Map<String, Object> arguments;

    protected McpDimension dimension;

    protected WorkflowTask workTask;

    protected McpConfig mcpConfig;

    protected McpResult<T> result;

    protected String uri;
}
