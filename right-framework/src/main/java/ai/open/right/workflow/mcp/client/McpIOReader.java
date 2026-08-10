package ai.open.right.workflow.mcp.client;

import java.io.Closeable;

public interface McpIOReader extends Closeable {

    public String readLine() throws Exception;
}
