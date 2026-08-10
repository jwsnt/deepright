package ai.open.right.workflow.mcp.client.rewrtier;

import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.McpResult;

import java.util.List;
import java.util.Map;

public interface McpRewriter {

    public static final String NAME = "globalMcpRewriter";

    public McpResult<List<Map<String, Object>>> toolsCall(McpContext<List<Map<String, Object>>> context) throws Exception;

    public McpResult<List<History>> promptGet(McpContext<List<History>> context) throws Exception;

    public McpResult<String> resourcesRead(McpContext<String> context) throws Exception;
}
