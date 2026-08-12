package ai.open.right.workflow.mcp.client.stdio;

import com.google.common.cache.CacheLoader;
import org.easymock.EasyMock;
import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.McpProtocol;
import ai.open.right.workflow.mcp.client.McpRequest;
import ai.open.right.workflow.mcp.client.McpResponse;
import com.google.common.util.concurrent.UncheckedExecutionException;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.StringUtils;

import java.io.File;
import java.io.IOException;
import java.util.Collections;
import java.util.concurrent.atomic.AtomicBoolean;

public class McpStdioClientTest {

    public static final String PYTHON = System.getProperty("PYTHON_HOME", System.getenv("PYTHON_HOME"));

    public static final String NPX = System.getProperty("NPX_HOME", System.getenv("NPX_HOME"));

    @Test
    public void testPython() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.PYTHON)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("SQLite Explorer", command, "src/test/resources/mcp/sqllite_server.py")) {
            String expect = "[{name=query_data, description=Execute SQL queries safely, inputSchema={properties={sql={type=string}}, required=[sql], type=object}, outputSchema={properties={result={type=string}}, required=[result], type=object, x-fastmcp-wrap-result=true}, _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect, stdClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            try {
                stdClient.toolsCall("query_data", Collections.singletonMap("sql", "SELECT 1"), ObjectBuilder.buildMcpDimensionWithMcpConfig());
            } catch (WorkflowException e) {
                Assert.assertEquals("Output validation error: {'A': 'B', 'C': {'D': 'E'}} is not of type 'string'", e.getMessage());
            }
            String expect2 = "[{name=summarize_request, description=Generate a prompt asking for a summary., arguments=[{name=text, required=true}], _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect2, stdClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            String expect3 = "[{name=get_schema, uri=schema://main, description=Provide the database schema as a resource, mimeType=text/plain, _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect3, stdClient.resourcesList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            Assert.assertEquals("[{uri=schema://main, mimeType=text/plain, text=A\n" + "B}]", stdClient.resourcesRead("schema://main", ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
        new File("sqlite3_database.db").deleteOnExit();
    }

    @Test(expected = CacheLoader.InvalidCacheLoadException.class)
    public void testPromptListAndNotFound() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.NPX + File.separator + "npx";
        try (McpStdioClient stdClient = new McpStdioClient("secure-filesystem-server", command, "-y", "@modelcontextprotocol/server-filesystem", "/", "/")) {
            Assert.assertEquals("", stdClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
    }

    @Test
    public void testPromptGet1() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.PYTHON)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("SQLite Explorer", command, "src/test/resources/mcp/sqllite_server.py")) {
            String expect = "[{name=summarize_request, description=Generate a prompt asking for a summary., arguments=[{name=text, required=true}], _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect, stdClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            Assert.assertEquals("{description=Generate a prompt asking for a summary., messages=[{role=user, content={type=text, text=Please summarize the following text:\n" + "\n" + "MyTable}}]}", stdClient.promptGet("summarize_request", Collections.singletonMap("text", "MyTable"), ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
    }

    @Test
    public void testPromptGet2() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.PYTHON)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("SQLite Explorer", command, "src/test/resources/mcp/sqllite_server.py")) {
            String expect = "[{name=summarize_request, description=Generate a prompt asking for a summary., arguments=[{name=text, required=true}], _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect, stdClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            Assert.assertEquals("{description=Generate a prompt asking for a summary., messages=[{role=user, content={type=text, text=Please summarize the following text:\n" + "\n" + "MyTable}}]}", stdClient.promptGet("summarize_request", Collections.singletonMap("text", "MyTable"), ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
    }

    @Test
    public void testNpx() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.NPX + File.separator + "npx";
        try (McpStdioClient stdClient = new McpStdioClient("secure-filesystem-server", command, "-y", "@modelcontextprotocol/server-filesystem@2025.1.14", "/", "/")) {
            Assert.assertNotNull(stdClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            Assert.assertTrue(stdClient.toolsCall("list_directory", Collections.singletonMap("path", "/"), ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString().contains("[FILE]"));
        }
    }

    @Test
    public void testEnv() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.PYTHON)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("SQLite Explorer", Collections.singletonMap("HELLO", "WORLD"), command, "src/test/resources/mcp/sqllite_server.py")) {
            String expect = "[{name=query_data, description=Execute SQL queries safely, inputSchema={properties={sql={type=string}}, required=[sql], type=object}, outputSchema={properties={result={type=string}}, required=[result], type=object, x-fastmcp-wrap-result=true}, _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect, stdClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            try {
                stdClient.toolsCall("query_data", Collections.singletonMap("sql", "SELECT 1"), ObjectBuilder.buildMcpDimensionWithMcpConfig());
            } catch (WorkflowException e) {
                Assert.assertEquals("Output validation error: {'A': 'B', 'C': {'D': 'E'}} is not of type 'string'", e.getMessage());
            }
            String expect2 = "[{name=summarize_request, description=Generate a prompt asking for a summary., arguments=[{name=text, required=true}], _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect2, stdClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            String expect3 = "[{name=get_schema, uri=schema://main, description=Provide the database schema as a resource, mimeType=text/plain, _meta={_fastmcp={tags=[]}}}]";
            Assert.assertEquals(expect3, stdClient.resourcesList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
            Assert.assertEquals("[{uri=schema://main, mimeType=text/plain, text=A\n" + "B}]", stdClient.resourcesRead("schema://main", ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
        new File("sqlite3_database.db").deleteOnExit();
    }

    @Test(expected = RuntimeException.class)
    public void testFail() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.PYTHON)) {
            return;
        }
        AtomicBoolean runner = new AtomicBoolean();
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("SQLite Explorer", Collections.singletonMap("HELLO", "WORLD"), command, "src/test/resources/mcp/sqllite_server.py") {
            protected void init(String name) throws Exception {
                throw new RuntimeException();
            }

            public void close() throws IOException {
                runner.set(true);
            }
        }) {
        } finally {
            Assert.assertTrue(runner.get());
            new File("sqlite3_database.db").deleteOnExit();
        }
    }

    @Test
    public void testToolsListAndNotFound() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("dynamic_resource", command, "src/test/resources/mcp/dynamic_resource.py")) {
            Assert.assertEquals("[]", stdClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
    }

    @Test(expected = UncheckedExecutionException.class)
    public void testToolsListAndException() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("dynamic_resource", command, "src/test/resources/mcp/dynamic_resource.py") {
            @Override
            protected McpResponse request(McpDimension mcpDimension, McpRequest request) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_TOOLS_LIST)) {
                    throw new RuntimeException("THIS IS A ERROR");
                }
                return super.request(request);
            }
        }) {
            stdClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        }
    }


    @Test
    public void testResourceTemplateListAndNotFound() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("multi_roles_prompt", command, "src/test/resources/mcp/multi_roles_prompt_script.py")) {
            Assert.assertEquals("[]", stdClient.resourcesTemplatesList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
    }

    @Test(expected = UncheckedExecutionException.class)
    public void testResourceTemplateListAndException() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("multi_roles_prompt", command, "src/test/resources/mcp/multi_roles_prompt_script.py") {
            @Override
            protected McpResponse request(McpDimension mcpDimension, McpRequest request) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_RESOURCES_TEMPLATES_LIST)) {
                    throw new RuntimeException("THIS IS A ERROR");
                }
                return super.request(request);
            }
        }) {
            stdClient.resourcesTemplatesList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        }
    }

    @Test
    public void testResourceListAndNotFound() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("multi_roles_prompt", command, "src/test/resources/mcp/multi_roles_prompt_script.py")) {
            Assert.assertEquals("[]", stdClient.resourcesList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).toString());
        }
    }

    @Test(expected = UncheckedExecutionException.class)
    public void testResourceTemplateAndException() throws Exception {
        if (!StringUtils.hasText(McpStdioClientTest.NPX)) {
            return;
        }
        String command = McpStdioClientTest.PYTHON + File.separator + "python3";
        try (McpStdioClient stdClient = new McpStdioClient("multi_roles_prompt", command, "src/test/resources/mcp/multi_roles_prompt_script.py") {
            @Override
            protected McpResponse request(McpDimension mcpDimension, McpRequest request) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_RESOURCES_LIST)) {
                    throw new RuntimeException("THIS IS A ERROR");
                }
                return super.request(request);
            }
        }) {
            stdClient.resourcesList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        }
    }

    @Test
    public void testSingleCommand() throws Exception {
        // pip3 install
        String command = "wikipedia-mcp";
        try (McpStdioClient stdClient = new McpStdioClient("Wiki", command)) {
            Assert.assertFalse(stdClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig()).isEmpty());
        }
    }
}
