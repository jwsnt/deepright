package ai.open.right.workflow.mcp.server;

public interface McpDistributor {

    public void distribute(McpRequest mcpRequest) throws Exception;
}
