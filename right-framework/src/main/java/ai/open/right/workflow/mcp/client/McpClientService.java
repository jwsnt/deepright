package ai.open.right.workflow.mcp.client;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;

import java.util.List;
import java.util.Map;

public interface McpClientService {

    public McpResult<List<Map<String, Object>>> toolsCall(String client, String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception;

    public McpResult<List<Map<String, Object>>> toolsCall(String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception;

    public List<ProviderFunCall> toolsList(String client, McpDimension mcpDimension) throws Exception;

    public McpResult<List<History>> promptGet(String client, String name, Map<String, Object> arguments, McpRuntime mcpRuntime, McpDimension mcpDimension) throws Exception;

    public McpResult<List<History>> promptGet(String client, String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception;

    public McpResult<List<History>> promptGet(String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception;

    public List<ProviderFunCall> promptList(String client, McpDimension mcpDimension) throws Exception;

    public List<ProviderFunCall> resourcesTemplatesList(String client, McpDimension mcpDimension) throws Exception;

    public List<ProviderFunCall> resourcesList(String client, McpDimension mcpDimension) throws Exception;

    public McpResult<String> resourcesRead(String client, String uri, McpRuntime mcpRuntime, McpDimension mcpDimension) throws Exception;

    public McpResult<String> resourcesRead(String client, String uri, McpDimension mcpDimension) throws Exception;
}