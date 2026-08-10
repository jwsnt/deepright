package ai.open.right.workflow.mcp.client;

import ai.open.right.workflow.mcp.client.dimension.McpDimension;

import java.io.Closeable;

public interface McpIOWriter extends Closeable {

    public void flush(McpDimension dimension) throws Exception;

    public void write(String content) throws Exception;
}
