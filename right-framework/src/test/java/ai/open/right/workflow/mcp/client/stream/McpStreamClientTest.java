package ai.open.right.workflow.mcp.client.stream;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.mcp.client.*;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import org.apache.http.HttpResponse;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.IOException;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class McpStreamClientTest {

    @Test
    public void test() throws Exception {
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.replay(client);
        new McpStreamClient(client, "name", "http", new HashMap<>()) {
            protected void init(String name) throws Exception {

            }
        };
    }

    @Test
    public void testClose() throws Exception {
        McpStreamClient mcpStreamClient = new McpStreamClient(null, "name", "http", new HashMap<>()) {
            protected void init(String name) throws Exception {

            }
        };
        mcpStreamClient.close();
    }

    @Test(expected = Exception.class)
    public void testInitException() throws Exception {
        new McpStreamClient(null, "name", "http", new HashMap<>()) {
            public void close() throws IOException {

            }
        };
    }

    @Test
    public void testCacheToolsList() throws Exception {
        McpStreamClient streamClient = new McpStreamClient(null, "name", "http", new HashMap<>()) {

            @Override
            protected McpResponse request(McpDimension dimension, McpRequest request, Boolean interrupt) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_TOOLS_LIST)) {
                    Map<String, Object> newone = new HashMap<>();
                    newone.put("tools", Arrays.asList("A", "B"));
                    McpResponse response = new McpResponse();
                    response.setResult(newone);
                    return response;
                }
                Map<String, Object> result = new HashMap<>();
                result.put("protocolVersion", McpClient.VERSION);
                McpResponse response = new McpResponse();
                response.setResult(result);
                return response;
            }

            public void close() throws IOException {

            }
        };
        streamClient.setHandler(new McpStreamHandler(null, null, null) {

            @Override
            protected void response(HttpRequestBase request) throws Exception {

            }

            @Override
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return null;
            }

            public String readLine() throws Exception {
                return "";
            }
        });
        List<Map<String, Object>> result1 = streamClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        List<Map<String, Object>> result2 = streamClient.toolsList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        Assert.assertEquals(result1, result2);
    }

    @Test
    public void testCachePromptList() throws Exception {
        McpStreamClient streamClient = new McpStreamClient(null, "name", "http", new HashMap<>()) {

            @Override
            protected McpResponse request(McpDimension dimension, McpRequest request, Boolean interrupt) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_PROMPTS_LIST)) {
                    Map<String, Object> newone = new HashMap<>();
                    newone.put("prompts", Arrays.asList("A", "B"));
                    McpResponse response = new McpResponse();
                    response.setResult(newone);
                    return response;
                }
                Map<String, Object> result = new HashMap<>();
                result.put("protocolVersion", McpClient.VERSION);
                McpResponse response = new McpResponse();
                response.setResult(result);
                return response;
            }

            public void close() throws IOException {

            }
        };
        streamClient.setHandler(new McpStreamHandler(null, null, null) {

            @Override
            protected void response(HttpRequestBase request) throws Exception {

            }

            @Override
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return null;
            }

            public String readLine() throws Exception {
                return "";
            }
        });
        List<Map<String, Object>> result1 = streamClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        List<Map<String, Object>> result2 = streamClient.promptList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        Assert.assertEquals(result1, result2);
    }

    @Test
    public void testCacheResourcesList() throws Exception {
        McpStreamClient streamClient = new McpStreamClient(null, "name", "http", new HashMap<>()) {

            @Override
            protected McpResponse request(McpDimension dimension, McpRequest request, Boolean interrupt) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_RESOURCES_LIST)) {
                    Map<String, Object> newone = new HashMap<>();
                    newone.put("resources", Arrays.asList("A", "B"));
                    McpResponse response = new McpResponse();
                    response.setResult(newone);
                    return response;
                }
                Map<String, Object> result = new HashMap<>();
                result.put("protocolVersion", McpClient.VERSION);
                McpResponse response = new McpResponse();
                response.setResult(result);
                return response;
            }

            public void close() throws IOException {

            }
        };
        streamClient.setHandler(new McpStreamHandler(null, null, null) {

            @Override
            protected void response(HttpRequestBase request) throws Exception {

            }

            @Override
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return null;
            }

            public String readLine() throws Exception {
                return "";
            }
        });
        List<Map<String, Object>> result1 = streamClient.resourcesList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        List<Map<String, Object>> result2 = streamClient.resourcesList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        Assert.assertEquals(result1, result2);
    }


    @Test
    public void testCacheResourcesTemplatesList() throws Exception {
        McpStreamClient streamClient = new McpStreamClient(null, "name", "http", new HashMap<>()) {

            @Override
            protected McpResponse request(McpDimension dimension, McpRequest request, Boolean interrupt) throws Exception {
                if (request.getProtocol().equals(McpProtocol.PROTOCOL_RESOURCES_TEMPLATES_LIST)) {
                    Map<String, Object> newone = new HashMap<>();
                    newone.put("resourceTemplates", Arrays.asList("A", "B"));
                    McpResponse response = new McpResponse();
                    response.setResult(newone);
                    return response;
                }
                Map<String, Object> result = new HashMap<>();
                result.put("protocolVersion", McpClient.VERSION);
                McpResponse response = new McpResponse();
                response.setResult(result);
                return response;
            }

            public void close() throws IOException {

            }
        };
        streamClient.setHandler(new McpStreamHandler(null, null, null) {

            @Override
            protected void response(HttpRequestBase request) throws Exception {

            }

            @Override
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return null;
            }

            public String readLine() throws Exception {
                return "";
            }
        });
        List<Map<String, Object>> result1 = streamClient.resourcesTemplatesList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        List<Map<String, Object>> result2 = streamClient.resourcesTemplatesList(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        Assert.assertEquals(result1, result2);
        Assert.assertNotNull(streamClient.getHandler());
        streamClient.setHandler(null);
        Assert.assertNull(streamClient.getHandler());
    }
}
