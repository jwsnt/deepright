package ai.open.right.workflow.flow.resource.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.resource.ResourceConfig;
import ai.open.right.workflow.flow.resource.ResourceFetcher;
import ai.open.right.workflow.flow.resource.ResourceRequest;
import ai.open.right.workflow.flow.resource.ResourceResponse;
import org.apache.http.Header;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.client.methods.HttpUriRequest;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.message.BasicHeader;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayInputStream;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.isNull;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

public class ResourceFetcherImplTest {

    private ResourceFetcherImpl resourceFetcher;
    private CloseableHttpAsyncClient httpClient;

    @BeforeEach
    public void setUp() {
        resourceFetcher = new ResourceFetcherImpl();
        httpClient = mock(CloseableHttpAsyncClient.class);
        resourceFetcher.setResource(httpClient);
        resourceFetcher.setRequest4resource(1000);
        resourceFetcher.setConnect4resource(2000);
        resourceFetcher.setSocket4resource(5000);
    }

    @Test
    public void testFetchGet() throws Exception {
        ResourceConfig config = new ResourceConfig();
        config.setTimeout(5000);
        
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("GET");
        Map<String, String> headers = new HashMap<>();
        headers.put("X-Test", "Value");
        request.setHeaders(headers);
        
        task.setObjectQuery(request);

        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        HttpEntity entity = mock(HttpEntity.class);
        Future<HttpResponse> future = mock(Future.class);

        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(5000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(entity);
        when(entity.getContent()).thenReturn(new ByteArrayInputStream("{\"result\":\"ok\"}".getBytes()));
        when(response.getAllHeaders()).thenReturn(new Header[]{new BasicHeader("Content-Type", "application/json")});

        ResourceResponse result = resourceFetcher.fetch(config, task);
        
        Assertions.assertEquals("{\"result\":\"ok\"}", result.getContent());
        Assertions.assertEquals("application/json", result.getHeaders().get("Content-Type"));
    }

    @Test
    public void testFetchPost() throws Exception {
        ResourceConfig config = new ResourceConfig();
        
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("POST");
        Map<String, Object> content = new HashMap<>();
        content.put("key", "value");
        request.setContent(content);
        
        task.setObjectQuery(request);

        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        HttpEntity entity = mock(HttpEntity.class);
        Future<HttpResponse> future = mock(Future.class);

        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(5000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(entity);
        when(entity.getContent()).thenReturn(new ByteArrayInputStream("ok".getBytes()));
        when(response.getAllHeaders()).thenReturn(new Header[0]);

        ResourceResponse result = resourceFetcher.fetch(config, task);
        
        Assertions.assertEquals("ok", result.getContent());
    }

    @Test
    public void testFetchNoConfigNoHeaders() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("GET");
        request.setHeaders(null);
        task.setObjectQuery(request);

        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        HttpEntity entity = mock(HttpEntity.class);
        Future<HttpResponse> future = mock(Future.class);

        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(5000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(entity);
        when(entity.getContent()).thenReturn(new ByteArrayInputStream("ok".getBytes()));
        when(response.getAllHeaders()).thenReturn(null);

        ResourceResponse result = resourceFetcher.fetch(null, task);
        Assertions.assertEquals("ok", result.getContent());
        Assertions.assertNull(result.getHeaders());
    }

    @Test
    public void testFetchErrorStatus() throws Exception {
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        task.setObjectQuery(request);

        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        HttpEntity entity = mock(HttpEntity.class);
        Future<HttpResponse> future = mock(Future.class);

        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(5000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(500);
        when(response.getEntity()).thenReturn(entity);
        when(entity.getContent()).thenReturn(new ByteArrayInputStream("error body".getBytes()));

        Assertions.assertThrows(WorkflowException.class, () -> {
            resourceFetcher.fetch(config, task);
        });
    }

    @Test
    public void testGetSetResource() {
        resourceFetcher.setResource(httpClient);
        Assertions.assertEquals(httpClient, resourceFetcher.getResource());
    }

    @Test
    public void testInitConfig() throws Exception {
        ResourceFetcherImpl.InitConfig initConfig = new ResourceFetcherImpl.InitConfig();
        initConfig.setResource(httpClient);
        Assertions.assertNotNull(initConfig.resourceFetcher());
        Assertions.assertEquals(httpClient, initConfig.getResource());
    }

    @Test
    public void testInitConfig_withNewConfigs() throws Exception {
        ResourceFetcherImpl.InitConfig initConfig = new ResourceFetcherImpl.InitConfig();
        initConfig.setResource(httpClient);
        initConfig.setRequest4resource(2000);
        initConfig.setConnect4resource(3000);
        initConfig.setSocket4resource(15000);
        ResourceFetcher fetcher = initConfig.resourceFetcher();
        Assertions.assertNotNull(fetcher);
        ResourceFetcherImpl impl = (ResourceFetcherImpl) fetcher;
        Assertions.assertEquals(Integer.valueOf(2000), impl.getRequest4resource());
        Assertions.assertEquals(Integer.valueOf(3000), impl.getConnect4resource());
        Assertions.assertEquals(Integer.valueOf(15000), impl.getSocket4resource());
    }

    @Test
    public void testInitConfig_defaultValues() {
        ResourceFetcherImpl.InitConfig initConfig = new ResourceFetcherImpl.InitConfig();
        // 默认值来自 @Value 注解
        Assertions.assertNull(initConfig.getRequest4resource());
        Assertions.assertNull(initConfig.getConnect4resource());
        Assertions.assertNull(initConfig.getSocket4resource());
    }

    // ---------- buildHttpResponse 全分支覆盖（新增3个配置） ----------

    @Test
    public void buildHttpResponse_withResourceConfigTimeout_usesConfigTimeout() throws Exception {
        ResourceConfig config = new ResourceConfig();
        config.setTimeout(5000);
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("GET");
        task.setObjectQuery(request);
        
        resourceFetcher.setRequest4resource(1000);
        resourceFetcher.setConnect4resource(2000);
        resourceFetcher.setSocket4resource(10000);
        
        HttpGet httpRequest = new HttpGet("http://test.com");
        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        Future<HttpResponse> future = mock(Future.class);
        
        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(5000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(null);
        
        HttpResponse result = resourceFetcher.buildHttpResponse(config, task, request, httpRequest);
        
        Assertions.assertSame(response, result);
        // 验证 RequestConfig 被设置
        org.apache.http.client.config.RequestConfig requestConfig = httpRequest.getConfig();
        Assertions.assertNotNull(requestConfig);
        Assertions.assertEquals(Integer.valueOf(1000), requestConfig.getConnectionRequestTimeout());
        Assertions.assertEquals(Integer.valueOf(2000), requestConfig.getConnectTimeout());
        Assertions.assertEquals(Integer.valueOf(5000), requestConfig.getSocketTimeout());
    }

    @Test
    public void buildHttpResponse_withoutResourceConfigTimeout_usesSocket4resource() throws Exception {
        ResourceConfig config = new ResourceConfig();
        config.setTimeout(null);
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("GET");
        task.setObjectQuery(request);
        
        resourceFetcher.setRequest4resource(1500);
        resourceFetcher.setConnect4resource(2500);
        resourceFetcher.setSocket4resource(12000);
        
        HttpGet httpRequest = new HttpGet("http://test.com");
        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        Future<HttpResponse> future = mock(Future.class);
        
        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(12000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(null);
        
        HttpResponse result = resourceFetcher.buildHttpResponse(config, task, request, httpRequest);
        
        Assertions.assertSame(response, result);
        // 验证 RequestConfig 被设置，socketTimeout 使用 socket4resource
        org.apache.http.client.config.RequestConfig requestConfig = httpRequest.getConfig();
        Assertions.assertNotNull(requestConfig);
        Assertions.assertEquals(Integer.valueOf(1500), requestConfig.getConnectionRequestTimeout());
        Assertions.assertEquals(Integer.valueOf(2500), requestConfig.getConnectTimeout());
        Assertions.assertEquals(Integer.valueOf(12000), requestConfig.getSocketTimeout());
    }

    @Test
    public void buildHttpResponse_nullConfig_usesSocket4resource() throws Exception {
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("GET");
        task.setObjectQuery(request);
        
        resourceFetcher.setRequest4resource(2000);
        resourceFetcher.setConnect4resource(3000);
        resourceFetcher.setSocket4resource(15000);
        
        HttpGet httpRequest = new HttpGet("http://test.com");
        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        Future<HttpResponse> future = mock(Future.class);
        
        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(15000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(null);
        
        HttpResponse result = resourceFetcher.buildHttpResponse(null, task, request, httpRequest);
        
        Assertions.assertSame(response, result);
        // 验证 RequestConfig 被设置，socketTimeout 使用 socket4resource
        org.apache.http.client.config.RequestConfig requestConfig = httpRequest.getConfig();
        Assertions.assertNotNull(requestConfig);
        Assertions.assertEquals(Integer.valueOf(2000), requestConfig.getConnectionRequestTimeout());
        Assertions.assertEquals(Integer.valueOf(3000), requestConfig.getConnectTimeout());
        Assertions.assertEquals(Integer.valueOf(15000), requestConfig.getSocketTimeout());
    }

    @Test
    public void buildHttpResponse_configTimeoutOverridesSocket4resource() throws Exception {
        ResourceConfig config = new ResourceConfig();
        config.setTimeout(8000);
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        request.setUrl("http://test.com");
        request.setMethod("GET");
        task.setObjectQuery(request);
        
        resourceFetcher.setRequest4resource(1000);
        resourceFetcher.setConnect4resource(2000);
        resourceFetcher.setSocket4resource(10000);
        
        HttpGet httpRequest = new HttpGet("http://test.com");
        HttpResponse response = mock(HttpResponse.class);
        StatusLine statusLine = mock(StatusLine.class);
        Future<HttpResponse> future = mock(Future.class);
        
        when(httpClient.execute(any(HttpRequestBase.class), isNull())).thenReturn(future);
        when(future.get(8000, TimeUnit.MILLISECONDS)).thenReturn(response);
        when(response.getStatusLine()).thenReturn(statusLine);
        when(statusLine.getStatusCode()).thenReturn(200);
        when(response.getEntity()).thenReturn(null);
        
        HttpResponse result = resourceFetcher.buildHttpResponse(config, task, request, httpRequest);
        
        Assertions.assertSame(response, result);
        // config timeout 覆盖 socket4resource
        org.apache.http.client.config.RequestConfig requestConfig = httpRequest.getConfig();
        Assertions.assertNotNull(requestConfig);
        Assertions.assertEquals(Integer.valueOf(8000), requestConfig.getSocketTimeout());
    }

    // ---------- buildHttpHeader 全分支覆盖（已去掉重复 put） ----------

    /** 覆盖 buildHttpHeader：resourceConfig == null, request.getHeaders() == null，只设置 Content-Type。 */
    @Test
    public void buildHttpHeader_nullConfig_nullRequestHeaders_onlyContentType() throws Exception {
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(null);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(null, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        Assertions.assertNull(httpRequest.getFirstHeader("Authorization"));
    }

    /** 覆盖 buildHttpHeader：resourceConfig == null, request.getHeaders() 为空 Map，只设置 Content-Type。 */
    @Test
    public void buildHttpHeader_nullConfig_emptyRequestHeaders_onlyContentType() throws Exception {
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(new HashMap<>());
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(null, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
    }

    /** 覆盖 buildHttpHeader：resourceConfig == null, request.getHeaders() 有允许的 headers，设置 Content-Type 和允许的 request headers。 */
    @Test
    public void buildHttpHeader_nullConfig_withAllowedRequestHeaders() throws Exception {
        ResourceRequest request = new ResourceRequest();
        Map<String, String> requestHeaders = new HashMap<>();
        requestHeaders.put("Authorization", "Bearer token123");
        requestHeaders.put("X-Custom", "value");
        request.setHeaders(requestHeaders);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(null, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        Assertions.assertEquals("Bearer token123", httpRequest.getFirstHeader("Authorization").getValue());
        Assertions.assertEquals("value", httpRequest.getFirstHeader("X-Custom").getValue());
    }

    /** 覆盖 buildHttpHeader：resourceConfig == null, request.getHeaders() 包含不允许的 headers（以 __ 开头或结尾），只设置允许的。 */
    @Test
    public void buildHttpHeader_nullConfig_withDisallowedRequestHeaders() throws Exception {
        ResourceRequest request = new ResourceRequest();
        Map<String, String> requestHeaders = new HashMap<>();
        requestHeaders.put("Authorization", "Bearer token123");
        requestHeaders.put("__Header", "disallowed-start");
        requestHeaders.put("Header__", "disallowed-end");
        requestHeaders.put("__Header__", "disallowed-both");
        requestHeaders.put("X-Custom", "value");
        request.setHeaders(requestHeaders);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(null, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        Assertions.assertEquals("Bearer token123", httpRequest.getFirstHeader("Authorization").getValue());
        Assertions.assertEquals("value", httpRequest.getFirstHeader("X-Custom").getValue());
        // 不允许的 headers 不应该被设置
        Assertions.assertNull(httpRequest.getFirstHeader("__Header"));
        Assertions.assertNull(httpRequest.getFirstHeader("Header__"));
        Assertions.assertNull(httpRequest.getFirstHeader("__Header__"));
    }

    /** 覆盖 buildHttpHeader：resourceConfig != null && !hasHeaders(), request.getHeaders() == null，只设置 Content-Type。 */
    @Test
    public void buildHttpHeader_configWithoutHeaders_nullRequestHeaders_onlyContentType() throws Exception {
        ResourceConfig config = new ResourceConfig();
        config.setHeaders(null);
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(null);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(config, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
    }

    /** 覆盖 buildHttpHeader：resourceConfig != null && !hasHeaders(), request.getHeaders() 有值，设置 Content-Type 和允许的 request headers。 */
    @Test
    public void buildHttpHeader_configWithoutHeaders_withRequestHeaders() throws Exception {
        ResourceConfig config = new ResourceConfig();
        config.setHeaders(new HashMap<>());
        ResourceRequest request = new ResourceRequest();
        Map<String, String> requestHeaders = new HashMap<>();
        requestHeaders.put("X-Request", "req-value");
        request.setHeaders(requestHeaders);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(config, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        Assertions.assertEquals("req-value", httpRequest.getFirstHeader("X-Request").getValue());
    }

    /** 覆盖 buildHttpHeader：resourceConfig != null && hasHeaders() && !getAutoCopy(), request.getHeaders() == null，只设置 Content-Type（不复制 config headers）。 */
    @Test
    public void buildHttpHeader_configWithHeaders_autoCopyFalse_nullRequestHeaders() throws Exception {
        ResourceConfig config = new ResourceConfig();
        Map<String, String> configHeaders = new HashMap<>();
        configHeaders.put("Authorization", "Bearer config-token");
        configHeaders.put("X-Config", "config-value");
        config.setHeaders(configHeaders);
        config.setAutoCopy(false);
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(null);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(config, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        // autoCopy=false，config headers 不应该被设置
        Assertions.assertNull(httpRequest.getFirstHeader("Authorization"));
        Assertions.assertNull(httpRequest.getFirstHeader("X-Config"));
    }

    /** 覆盖 buildHttpHeader：resourceConfig != null && hasHeaders() && getAutoCopy(), request.getHeaders() == null，设置 Content-Type 和 config headers。 */
    @Test
    public void buildHttpHeader_configWithHeaders_autoCopyTrue_nullRequestHeaders() throws Exception {
        ResourceConfig config = new ResourceConfig();
        Map<String, String> configHeaders = new HashMap<>();
        configHeaders.put("Authorization", "Bearer config-token");
        configHeaders.put("X-Config", "config-value");
        config.setHeaders(configHeaders);
        config.setAutoCopy(true);
        ResourceRequest request = new ResourceRequest();
        request.setHeaders(null);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(config, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        Assertions.assertEquals("Bearer config-token", httpRequest.getFirstHeader("Authorization").getValue());
        Assertions.assertEquals("config-value", httpRequest.getFirstHeader("X-Config").getValue());
    }

    /** 覆盖 buildHttpHeader：resourceConfig != null && hasHeaders() && getAutoCopy(), request.getHeaders() 有允许的 headers，设置 Content-Type、config headers 和允许的 request headers（后覆盖前）。 */
    @Test
    public void buildHttpHeader_configWithHeaders_autoCopyTrue_requestWithAllowedHeaders() throws Exception {
        ResourceConfig config = new ResourceConfig();
        Map<String, String> configHeaders = new HashMap<>();
        configHeaders.put("Authorization", "Bearer config-token");
        configHeaders.put("X-Config", "config-value");
        configHeaders.put("X-Shared", "config-shared");
        config.setHeaders(configHeaders);
        config.setAutoCopy(true);
        ResourceRequest request = new ResourceRequest();
        Map<String, String> requestHeaders = new HashMap<>();
        requestHeaders.put("Authorization", "Bearer request-token");
        requestHeaders.put("X-Request", "request-value");
        requestHeaders.put("X-Shared", "request-shared");
        request.setHeaders(requestHeaders);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(config, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        // request 覆盖 config（允许的 headers）
        Assertions.assertEquals("Bearer request-token", httpRequest.getFirstHeader("Authorization").getValue());
        Assertions.assertEquals("request-shared", httpRequest.getFirstHeader("X-Shared").getValue());
        // config 独有的
        Assertions.assertEquals("config-value", httpRequest.getFirstHeader("X-Config").getValue());
        // request 独有的
        Assertions.assertEquals("request-value", httpRequest.getFirstHeader("X-Request").getValue());
    }

    /** 覆盖 buildHttpHeader：resourceConfig != null && hasHeaders() && getAutoCopy(), request.getHeaders() 包含不允许的 headers，只设置允许的。 */
    @Test
    public void buildHttpHeader_configWithHeaders_autoCopyTrue_requestWithDisallowedHeaders() throws Exception {
        ResourceConfig config = new ResourceConfig();
        Map<String, String> configHeaders = new HashMap<>();
        configHeaders.put("Authorization", "Bearer config-token");
        config.setHeaders(configHeaders);
        config.setAutoCopy(true);
        ResourceRequest request = new ResourceRequest();
        Map<String, String> requestHeaders = new HashMap<>();
        requestHeaders.put("Authorization", "Bearer request-token");
        requestHeaders.put("__Disallowed", "disallowed-start");
        requestHeaders.put("Disallowed__", "disallowed-end");
        requestHeaders.put("X-Allowed", "allowed-value");
        request.setHeaders(requestHeaders);
        HttpGet httpRequest = new HttpGet("http://test.com");
        
        HttpUriRequest result = resourceFetcher.buildHttpHeader(config, ObjectBuilder.buildWorkflowTask(), request, httpRequest);
        
        Assertions.assertSame(httpRequest, result);
        Assertions.assertEquals("application/json", httpRequest.getFirstHeader("Content-Type").getValue());
        // config header
        Assertions.assertEquals("Bearer request-token", httpRequest.getFirstHeader("Authorization").getValue());
        // request 允许的 header（覆盖 config）
        Assertions.assertEquals("Bearer request-token", httpRequest.getFirstHeader("Authorization").getValue());
        Assertions.assertEquals("allowed-value", httpRequest.getFirstHeader("X-Allowed").getValue());
        // 不允许的 headers 不应该被设置
        Assertions.assertNull(httpRequest.getFirstHeader("__Disallowed"));
        Assertions.assertNull(httpRequest.getFirstHeader("Disallowed__"));
    }

    // ---------- allowedHeader 全分支覆盖 ----------

    /** 测试子类用于访问 protected 方法 */
    private static class TestResourceFetcherImpl extends ResourceFetcherImpl {
        public Boolean testAllowedHeader(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, String key) throws Exception {
            return this.allowedHeader(resourceConfig, workTask, request, key);
        }
        
        public ResourceResponse testBuildResourceResponse(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, HttpResponse response) throws Exception {
            return this.buildResourceResponse(resourceConfig, workTask, request, response);
        }
        
        public String testBuildContent(ResourceConfig resourceConfig, WorkflowTask workTask, ResourceRequest request, String content) throws Exception {
            return this.buildContent(resourceConfig, workTask, request, content);
        }
    }

    @Test
    public void allowedHeader_normalKey_returnsTrue() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        
        Assertions.assertTrue(fetcher.testAllowedHeader(config, task, request, "Authorization"));
        Assertions.assertTrue(fetcher.testAllowedHeader(config, task, request, "Content-Type"));
        Assertions.assertTrue(fetcher.testAllowedHeader(config, task, request, "X-Custom-Header"));
        Assertions.assertTrue(fetcher.testAllowedHeader(config, task, request, "normal"));
    }

    @Test
    public void allowedHeader_startsWithHeaderKey_returnsFalse() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__Header"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__header"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__HEADER"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__Custom"));
    }

    @Test
    public void allowedHeader_endsWithHeaderKey_returnsFalse() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "Header__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "header__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "HEADER__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "Custom__"));
    }

    @Test
    public void allowedHeader_startsAndEndsWithHeaderKey_returnsFalse() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__Header__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__header__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__HEADER__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__"));
    }

    @Test
    public void allowedHeader_caseInsensitive() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        
        // 大小写不敏感：__ 和 __ 应该匹配
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__Header"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "__HEADER"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "Header__"));
        Assertions.assertFalse(fetcher.testAllowedHeader(config, task, request, "HEADER__"));
        
        // 中间包含 __ 但不在开头或结尾，应该允许
        Assertions.assertTrue(fetcher.testAllowedHeader(config, task, request, "Header__Middle"));
        Assertions.assertTrue(fetcher.testAllowedHeader(config, task, request, "Normal__Value"));
    }

    @Test
    public void buildResourceResponse_setsContentAndHeaders() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        HttpResponse response = mock(HttpResponse.class);
        HttpEntity entity = mock(HttpEntity.class);
        when(response.getEntity()).thenReturn(entity);
        when(entity.getContent()).thenReturn(new ByteArrayInputStream("body".getBytes()));
        when(response.getAllHeaders()).thenReturn(new Header[]{new BasicHeader("X-Header", "value")});
        
        ResourceResponse result = fetcher.testBuildResourceResponse(config, task, request, response);
        
        Assertions.assertNotNull(result);
        Assertions.assertEquals("body", result.getContent());
        Assertions.assertEquals("value", result.getHeaders().get("X-Header"));
    }

    @Test
    public void buildResourceResponse_nullEntity_setsEmptyContent() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        HttpResponse response = mock(HttpResponse.class);
        when(response.getEntity()).thenReturn(null);
        when(response.getAllHeaders()).thenReturn(new Header[0]);
        
        ResourceResponse result = fetcher.testBuildResourceResponse(config, task, request, response);
        
        Assertions.assertNotNull(result);
        Assertions.assertEquals("", result.getContent());
        Assertions.assertNull(result.getHeaders());
    }

    @Test
    public void buildResourceResponse_nullHeaders_setsContentOnly() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        HttpResponse response = mock(HttpResponse.class);
        HttpEntity entity = mock(HttpEntity.class);
        when(response.getEntity()).thenReturn(entity);
        when(entity.getContent()).thenReturn(new ByteArrayInputStream("ok".getBytes()));
        when(response.getAllHeaders()).thenReturn(null);
        
        ResourceResponse result = fetcher.testBuildResourceResponse(config, task, request, response);
        
        Assertions.assertNotNull(result);
        Assertions.assertEquals("ok", result.getContent());
        Assertions.assertNull(result.getHeaders());
    }

    // ---------- buildContent 全分支覆盖 ----------

    @Test
    public void buildContent_returnsContentAsIs() throws Exception {
        TestResourceFetcherImpl fetcher = new TestResourceFetcherImpl();
        ResourceConfig config = new ResourceConfig();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        ResourceRequest request = new ResourceRequest();
        
        Assertions.assertEquals("raw", fetcher.testBuildContent(config, task, request, "raw"));
        Assertions.assertEquals("", fetcher.testBuildContent(config, task, request, ""));
    }
}
