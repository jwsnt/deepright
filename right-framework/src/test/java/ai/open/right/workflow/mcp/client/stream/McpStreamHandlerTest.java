package ai.open.right.workflow.mcp.client.stream;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import com.google.common.collect.ImmutableMap;
import org.apache.http.Header;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Future;

public class McpStreamHandlerTest {

    @Test
    public void testResponse() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        HttpResponse httpResponse = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(httpResponse.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(httpResponse.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("ABC".getBytes())).anyTimes();
        Header header = EasyMock.createMock(Header.class);
        EasyMock.expect(header.getValue()).andReturn("xxxxxx").anyTimes();
        EasyMock.expect(httpResponse.getFirstHeader("Mcp-Session-Id")).andReturn(header).anyTimes();
        EasyMock.replay(client, entity, header, httpResponse, statusLine);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return httpResponse;
            }
        };
        handler.response(new HttpPost("http://x.y.z"));
        Assert.assertEquals("ABC", handler.readLine());
        Assert.assertNull(handler.response);
        EasyMock.verify(client, entity, header, httpResponse, statusLine);
    }

    @Test
    public void testResponseWithJson() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        HttpResponse httpResponse = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(httpResponse.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(httpResponse.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("{\"ABC\":\"XYZ\"}".getBytes())).anyTimes();
        Header header = EasyMock.createMock(Header.class);
        EasyMock.expect(header.getValue()).andReturn("xxxxxx").anyTimes();
        EasyMock.expect(httpResponse.getFirstHeader("Mcp-Session-Id")).andReturn(header).anyTimes();
        EasyMock.replay(client, entity, header, httpResponse, statusLine);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return httpResponse;
            }
        };
        handler.response(new HttpPost("http://x.y.z"));
        Assert.assertEquals("{\"ABC\":\"XYZ\"}", handler.readLine());
        EasyMock.verify(client, entity, header, httpResponse, statusLine);
    }

    @Test
    public void testResponseWithJsonStream() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        HttpResponse httpResponse = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(httpResponse.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(httpResponse.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("ABCDE{\"ABC\":\"XYZ\"}".getBytes())).anyTimes();
        Header header = EasyMock.createMock(Header.class);
        EasyMock.expect(header.getValue()).andReturn("xxxxxx").anyTimes();
        EasyMock.expect(httpResponse.getFirstHeader("Mcp-Session-Id")).andReturn(header).anyTimes();
        EasyMock.replay(client, entity, header, httpResponse, statusLine);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected HttpResponse execute(HttpRequestBase request) throws Exception {
                return httpResponse;
            }
        };
        handler.response(new HttpPost("http://x.y.z"));
        Assert.assertEquals("{\"ABC\":\"XYZ\"}", handler.readLine());
        EasyMock.verify(client, entity, header, httpResponse, statusLine);
    }

    @Test
    public void testWriter() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("ABC".getBytes())).anyTimes();
        EasyMock.replay(client, entity, response, statusLine);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected void response(HttpRequestBase request) throws Exception {
                Assert.assertEquals(request.getFirstHeader("Content-Type").getValue(), McpStreamHandler.CONTENT_TYPE);
                Assert.assertEquals(request.getFirstHeader("Accept").getValue(), McpStreamHandler.ACCEPT);
                Assert.assertEquals(request.getFirstHeader("Mcp-Session-Id").getValue(), "xxxxxxx");
            }
        };
        handler.write("ABC");
        handler.session = "xxxxxxx";
        handler.flush(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        Assert.assertNull(handler.request);
        EasyMock.verify(client, entity, response, statusLine);
    }

    @Test
    public void testWriterWithHeaders() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        headers.put("A1", "B1");
        headers.put("A2", "B2");
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("ABC".getBytes())).anyTimes();
        EasyMock.replay(client, entity, response, statusLine);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected void response(HttpRequestBase request) throws Exception {
                Assert.assertEquals(headers.get("A1"), "B1");
                Assert.assertEquals(headers.get("A2"), "B2");
            }
        };
        handler.write("ABC");
        handler.session = "xxxxxxx";
        handler.flush(ObjectBuilder.buildMcpDimensionWithMcpConfig());
        Assert.assertNull(handler.request);
        EasyMock.verify(client, entity, response, statusLine);
    }

    @Test
    public void testWriterWithAdditionalHeaders() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        headers.put("A1", "B1");
        headers.put("A2", "B2");
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("ABC".getBytes())).anyTimes();
        EasyMock.replay(client, entity, response, statusLine);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected void response(HttpRequestBase request) throws Exception {
                Assert.assertEquals(request.getHeaders("A1")[0].getValue(), "B1");
                Assert.assertEquals(request.getHeaders("A2")[0].getValue(), "B2");
                Assert.assertEquals(request.getHeaders("HELLO")[0].getValue(), "WORLD");
            }
        };
        handler.write("ABC");
        handler.session = "xxxxxxx";
        McpDimension dimension = McpDimension.builder()
                .headers(ImmutableMap.of("HELLO","WORLD"))
                .build();
        handler.flush(dimension);
        Assert.assertNull(handler.request);
        EasyMock.verify(client, entity, response, statusLine);
    }

    @Test
    public void testExecute() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z");
        Future<HttpResponse> future = EasyMock.createMock(Future.class);
        HttpResponse httpResponse = EasyMock.createMock(HttpResponse.class);
        EasyMock.expect(future.get()).andReturn(httpResponse);
        HttpPost post = new HttpPost("ABC");
        EasyMock.expect(client.execute(post, null)).andReturn(future).anyTimes();
        EasyMock.replay(client, future, httpResponse);
        Assert.assertNotNull(handler.execute(post));
        EasyMock.verify(client, future, httpResponse);
    }

    @Test
    public void testClose() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected void response(HttpRequestBase request) throws Exception {
                Assert.assertEquals(request.getFirstHeader("Content-Type").getValue(), McpStreamHandler.CONTENT_TYPE);
                Assert.assertEquals(request.getFirstHeader("Accept").getValue(), McpStreamHandler.ACCEPT);
                Assert.assertEquals(request.getFirstHeader("Mcp-Session-Id").getValue(), "xxxxxxx");
            }
        };
        EasyMock.replay(client);
        handler.session = "xxxxxxx";
        handler.close();
        EasyMock.verify(client);
    }

    @Test(expected = IOException.class)
    public void testCloseWithException() throws Exception {
        Map<String, String> headers = new HashMap<String, String>();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        McpStreamHandler handler = new McpStreamHandler(client, headers, "http://x.y.z") {
            protected void response(HttpRequestBase request) throws Exception {
                throw new RuntimeException("ABC");
            }
        };
        EasyMock.replay(client);
        handler.session = "xxxxxxx";
        handler.close();
        EasyMock.verify(client);
    }
    @Test
    public void testSessionMissing() {
        McpStreamHandler handler = new McpStreamHandler(null, null, "http://x");
        handler.session(EasyMock.createMock(HttpResponse.class));
        Assert.assertNull(handler.session);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testReadLineEmpty() throws Exception {
        McpStreamHandler handler = new McpStreamHandler(null, null, "http://x");
        handler.response = null;
        handler.readLine();
    }
}
