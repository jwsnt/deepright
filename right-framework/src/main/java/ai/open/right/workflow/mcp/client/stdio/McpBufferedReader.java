package ai.open.right.workflow.mcp.client.stdio;

import ai.open.right.workflow.mcp.client.McpIOReader;

import java.io.BufferedReader;
import java.io.IOException;

public class McpBufferedReader implements McpIOReader {

    protected final BufferedReader reader;

    public McpBufferedReader(BufferedReader reader) {
        this.reader = reader;
    }

    @Override
    public String readLine() throws Exception {
        return this.reader.readLine();
    }

    @Override
    public void close() throws IOException {
        this.reader.close();
    }
}
