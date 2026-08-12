package ai.open.right.workflow.mcp.client;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.stdio.McpBufferedReader;
import ai.open.right.workflow.mcp.client.stdio.McpStdioClient;
import ai.open.right.workflow.mcp.client.stdio.McpStdioClientTest;
import com.fasterxml.jackson.core.JsonParseException;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.StringUtils;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.InputStreamReader;
import java.util.Collections;
import java.util.concurrent.atomic.AtomicInteger;

public class McpClientTest {

    public static final String NPX = System.getProperty("NPX_HOME", System.getenv("NPX_HOME"));

    @Test(expected = WorkflowException.class)
    public void testErrorStream() throws Exception {
        if (!StringUtils.hasText(McpClientTest.NPX)) {
            return;
        }
        String command = McpClientTest.NPX + File.separator + "npx";
        try (McpStdioClient stdClient = new McpStdioClient("secure-filesystem-server", command, "-y", "@modelcontextprotocol/server-filesystem@2025.1.14", "/", "/")) {
            stdClient.stdInput = new McpBufferedReader(new BufferedReader(new InputStreamReader(new ByteArrayInputStream("HELLO".getBytes()))));
            stdClient.toolsCall("list_directory", Collections.singletonMap("path", "/"), ObjectBuilder.buildMcpDimensionWithMcpConfig());
        } catch (WorkflowException e) {
            Assert.assertEquals("Mcp server response cannot be parsed, PROTOCOL_TOOLS_CALL: HELLO", e.getMessage());
            throw e;
        }
    }

    @Test(expected = Exception.class)
    public void testParser() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.PYTHON)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        new McpStdioClient("SQLite Explorer", command, "src/test/resources/mcp/sqllite_server.py") {

            private int json = 0;

            @Override
            protected McpResponse response(McpRequest request, Boolean interrupt) throws Exception {
                Exception e = json == 0 ? new JsonParseException("OK") : new WorkflowException();
                json++;
                throw e;
            }
        };
    }

    @Test
    public void testPromptGetEmptyArgs() throws Exception {
        McpClient client = new McpClient() {
            @Override
            protected McpResponse request(McpDimension dimension, McpRequest request) {
                Assert.assertNotNull(((java.util.Map) request.getParams()).get("arguments"));
                return new McpResponse();
            }
        };
        client.promptGet("NAME", null, null);
    }

    @Test
    public void testResponseThrowsWhenInterruptedBeforeRead() throws Exception {
        McpIOReader reader = new McpIOReader() {
            @Override
            public String readLine() {
                Assert.fail("readLine must not run when thread is already interrupted");
                return null;
            }

            @Override
            public void close() {
            }
        };
        McpClient client = new McpClient() {
        };
        client.setStdInput(reader);
        Thread.currentThread().interrupt();
        try {
            client.response(new McpRequest(McpProtocol.PROTOCOL_TOOLS_LIST), false);
            Assert.fail("expected InterruptedException");
        } catch (InterruptedException e) {
            Assert.assertEquals("MCP response reading interrupted", e.getMessage());
        } finally {
            Thread.interrupted();
        }
    }

    @Test
    public void testResponseThrowsWhenInterruptedAfterNonJsonLine() throws Exception {
        AtomicInteger readCount = new AtomicInteger();
        McpIOReader reader = new McpIOReader() {
            @Override
            public String readLine() {
                Assert.assertEquals(1, readCount.incrementAndGet());
                // 读完一行非 JSON 后下一轮循环开头会检查中断；在此处打断当前线程
                Thread.currentThread().interrupt();
                return "stderr-noise";
            }

            @Override
            public void close() {
            }
        };
        McpClient client = new McpClient() {
        };
        client.setStdInput(reader);
        try {
            client.response(new McpRequest(McpProtocol.PROTOCOL_TOOLS_LIST), false);
            Assert.fail("expected InterruptedException");
        } catch (InterruptedException e) {
            Assert.assertEquals("MCP response reading interrupted", e.getMessage());
            Assert.assertEquals(1, readCount.get());
        } finally {
            Thread.interrupted();
        }
    }

    @Test
    public void testStdOutputStdInputGetterSetter() {
        McpIOWriter writer = new McpIOWriter() {
            @Override
            public void flush(McpDimension dimension) {
            }

            @Override
            public void write(String content) {
            }

            @Override
            public void close() {
            }
        };
        McpIOReader reader = new McpIOReader() {
            @Override
            public String readLine() {
                return null;
            }

            @Override
            public void close() {
            }
        };
        McpClient client = new McpClient() {
        };
        client.setStdOutput(writer);
        client.setStdInput(reader);
        Assert.assertSame(writer, client.getStdOutput());
        Assert.assertSame(reader, client.getStdInput());
    }
}
