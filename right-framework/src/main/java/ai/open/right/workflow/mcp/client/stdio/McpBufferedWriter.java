package ai.open.right.workflow.mcp.client.stdio;

import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpIOWriter;

import java.io.BufferedWriter;
import java.io.IOException;

public class McpBufferedWriter implements McpIOWriter {

    protected final BufferedWriter writer;

    public McpBufferedWriter(BufferedWriter writer) {
        this.writer = writer;
    }

    @Override
    public void flush(McpDimension dimension) throws Exception {
        this.writer.flush();
    }

    @Override
    public void write(String content) throws Exception {
        this.writer.write(content);
    }

    @Override
    public void close() throws IOException {
        this.writer.close();
    }
}
