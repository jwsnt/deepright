package ai.open.right.workflow.mcp.client.rewrtier;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpResult;

import java.util.List;
import java.util.Map;

// MCP响应重写（请求后）
public interface McpRewriteService {

    public McpResult<List<Map<String, Object>>> toolsCall(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask, McpResult<List<Map<String, Object>>> result) throws Exception;

    public McpResult<List<History>> promptGet(McpDimension dimension, Map<String, Object> arguments, WorkflowTask workTask, McpResult<List<History>> result) throws Exception;

    public McpResult<String> resourcesRead(McpDimension dimension, String uri, WorkflowTask workTask, McpResult<String> result) throws Exception;
}
