package ai.open.right.workflow.flow.llm.provider.google;

import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.util.concurrent.Future;

public class VertexTokenExchangeTest {

    @Test
    public void testInitWithAppId() throws Exception {
        VertexTokenExchange vertexTokenExchange = new VertexTokenExchange();
        vertexTokenExchange.setRemote("http://12345.com?appid=#app_id");
        vertexTokenExchange.setAppId("OK");
        vertexTokenExchange.init();
        Assert.assertEquals("http://12345.com?appid=OK", vertexTokenExchange.getUrl());
    }

    @Test
    public void testInitWithProject() throws Exception {
        VertexTokenExchange vertexTokenExchange = new VertexTokenExchange();
        vertexTokenExchange.setRemote("http://12345.com?appid=#app_id");
        vertexTokenExchange.setProject("PO");
        vertexTokenExchange.init();
        Assert.assertEquals("http://12345.com?appid=PO", vertexTokenExchange.getUrl());
    }

    @Test
    public void testExchange() throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("{\"data\":\"OK\"}".getBytes())).anyTimes();
        EasyMock.replay(response, entity, statusLine);
        VertexTokenExchange vertexTokenExchange = new VertexTokenExchange() {
            protected HttpResponse response(HttpRequestBase httpRequestBase) throws Exception {
                return response;
            }
        };
        vertexTokenExchange.setRemote("http://12345.com?appid=#app_id");
        vertexTokenExchange.setProject("Project");
        vertexTokenExchange.setAppId("OK");
        vertexTokenExchange.init();
        Assert.assertEquals("OK", vertexTokenExchange.exchange());
        EasyMock.verify(response, entity, statusLine);
    }

    @Test
    public void testExchangeWithNull2() throws Exception {
        VertexTokenExchange vertexTokenExchange = new VertexTokenExchange();
        vertexTokenExchange.setRemote("http://hello.com");
        vertexTokenExchange.init();
        Assert.assertNull(vertexTokenExchange.exchange());
    }

    @Test
    public void testResponse() throws Exception {
        HttpGet httpGet = new HttpGet("http://12345.com");
        HttpResponse httpResponse = EasyMock.createMock(HttpResponse.class);
        Future future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andReturn(httpResponse).anyTimes();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.expect(client.execute(httpGet, null)).andReturn(future).anyTimes();
        EasyMock.replay(httpResponse, future, client);
        VertexTokenExchange vertexTokenExchange = new VertexTokenExchange();
        vertexTokenExchange.setOther(client);
        Assert.assertNotNull(vertexTokenExchange.response(httpGet));
        EasyMock.verify(httpResponse, future, client);
    }

    @Test
    public void testWithExchangeAndNullUrl() throws Exception {
        VertexTokenExchange vertexTokenExchange = new VertexTokenExchange();
        vertexTokenExchange.init();
        Assert.assertNull(vertexTokenExchange.exchange());
    }
    @Test(expected = IllegalArgumentException.class)
    public void testExchangeFail() throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.expect(status.getStatusCode()).andReturn(500).anyTimes();
        EasyMock.replay(response, status);
        VertexTokenExchange exchange = new VertexTokenExchange() {
            @Override protected HttpResponse response(HttpRequestBase r) { return response; }
        };
        exchange.setUrl("http://x");
        exchange.exchange();
    }

    @Test
    public void testInitNulls() throws Exception {
        VertexTokenExchange exchange = new VertexTokenExchange();
        exchange.setProject(null);
        exchange.setAppId(null);
        exchange.init();
        Assert.assertNull(exchange.getUrl());
    }
}
