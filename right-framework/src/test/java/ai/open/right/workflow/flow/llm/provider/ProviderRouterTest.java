package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.workflow.flow.llm.*;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.concurrent.FutureCallback;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClientBuilder;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;

public class ProviderRouterTest {

    @Test
    public void testBuildRequest() throws Exception {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return null;
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "HELLO WORLD";
            }
        };
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        router.setMaxSize(1024);
        router.setMaxSizeRate(1.0);
        Assert.assertNotNull(router.getHttpClientConfig());
        router.setEventListenerService(new EventListenerServiceImpl());
        router.setQueueTimeout(1026);
        router.setQueue(1027);
        router.setTimeoutRate(2d);
        router.setDiscard(600000);
        router.setTimeout(1026);
        Assert.assertEquals(Integer.valueOf(1026), router.getQueueTimeout());
        Assert.assertEquals(Integer.valueOf(1027), router.getQueue());
        Assert.assertEquals(Integer.valueOf(600000), router.getDiscard());
        Assert.assertEquals(Integer.valueOf(1026), router.getTimeout());
        ProviderRequest request = new ProviderRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        request.setFunCallTimeout(1000);
        HttpPost httpPost = router.buildRequest(request, new LLMConfig(), "https://api.example.com/v1/chat/completions");
        Assert.assertNotNull(httpPost);
        Assert.assertNotNull(httpPost.getEntity());
        Assert.assertEquals("HELLO WORLD", request.getProviderData().getRequest());
    }

    /**
     * buildRequest：limitSize=(int)(maxSize*maxSizeRate)；actualSize<=limitSize 时通过，并设置 entity、调用 request.appendRequest(entity)
     */
    @Test
    public void testBuildRequestRespectsMaxSizeAndMaxRate() throws Exception {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return null;
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "ab";
            }
        };
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        router.setMaxSize(100);
        router.setMaxSizeRate(0.5);
        router.setTimeoutRate(2d);
        router.setTimeout(1000);
        ProviderRequest request = new ProviderRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        request.setFunCallTimeout(1000);
        HttpPost httpPost = router.buildRequest(request, new LLMConfig(), "http://u");
        Assert.assertNotNull(httpPost);
        Assert.assertNotNull(httpPost.getEntity());
        Assert.assertEquals("ab", request.getProviderData().getRequest());
    }

    /**
     * buildRequest：actualSize > limitSize 时 Assert.isTrue 抛出 IllegalArgumentException
     */
    @Test(expected = IllegalArgumentException.class)
    public void testBuildRequestBodyExceedsLimitThrows() throws Exception {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return null;
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "xxxxxxxxxx";
            }
        };
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        router.setMaxSize(5);
        router.setTimeoutRate(2d);
        router.setMaxSizeRate(1.0);
        router.setTimeout(1000);
        ProviderRequest request = new ProviderRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        request.setFunCallTimeout(1000);
        router.buildRequest(request, new LLMConfig(), "http://u");
    }

    /**
     * ProviderRouter.maxSizeRate getter/setter 及 InitConfig @Value("${provider.size.max.rate:1.5}") 默认 1.5
     */
    @Test
    public void testMaxSizeRateGetterSetterAndInitConfigDefault() {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return null;
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        Assert.assertNull(router.getMaxSizeRate());
        router.setMaxSizeRate(1.5);
        Assert.assertEquals(Double.valueOf(1.5), router.getMaxSizeRate());
        ProviderRouter.ProviderRouterInitConfig initConfig = new ProviderRouter.ProviderRouterInitConfig();
        Assert.assertEquals(Double.valueOf(1.5), initConfig.getMaxSizeRate());
        initConfig.setMaxSizeRate(0.8);
        Assert.assertEquals(Double.valueOf(0.8), initConfig.getMaxSizeRate());
    }

    @Test
    public void testBuildRouter1() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMaxError(1000);
        req.setToken("Token");
        req.setStream(true);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {

            }
        };
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return reader;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "https://www.w3.org/";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        router.setStream(client);
        router.setOnce(client);
        router.setTimeout(5000);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        router.setMaxSize(1024);
        router.setTimeoutRate(2d);
        router.setMaxSizeRate(1.0);
        Assert.assertNotNull(router.getHttpClientConfig());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.setUrl("https://www.w3.org/");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(2000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        request.appendRequest("{}");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        eventListenerManager.listen(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(callback, status, response, request, entity, llmCallback, eventListenerManager);
        router.setEventListenerService(eventListenerManager);
        router.setExecutorService(Executors.newFixedThreadPool(1));
        Assert.assertNotNull(router.getExecutorService());
        router.route(request, new LLMConfig(), llmCallback);
        EasyMock.verify(callback, status, response, entity, request, llmCallback, eventListenerManager);
        client.close();
    }

    @Test
    public void testBuildRouter2() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(false);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setMaxError(1000);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {

            }
        };
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return reader;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "https://www.w3.org/";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        router.setTimeoutRate(2d);
        router.setStream(client);
        router.setOnce(client);
        router.setTimeout(5000);
        router.setBuffer(1024);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        router.setMaxSize(1024);
        router.setMaxSizeRate(1.0);
        Assert.assertNotNull(router.getHttpClientConfig());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.setUrl("https://www.w3.org/");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(null).anyTimes();
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(2000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        request.appendRequest("{}");
        Assert.assertEquals(Integer.valueOf(1024), router.getBuffer());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        eventListenerManager.listen(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(callback, status, response, request, entity, llmCallback, eventListenerManager);
        router.setEventListenerService(eventListenerManager);
        router.setExecutorService(Executors.newFixedThreadPool(1));
        router.route(request, new LLMConfig(), llmCallback);
        EasyMock.verify(callback, status, response, entity, request, llmCallback, eventListenerManager);
        client.close();
    }

    @Test(expected = Exception.class)
    public void testBuildRouterFailed() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(false);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader<ProviderRequest> reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                throw new RuntimeException();
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "http://abc123456abc.efg.com";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        router.setStream(client);
        router.setOnce(client);
        router.setTimeout(1);
        router.setMaxSize(1024);
        router.setMaxSizeRate(1.0);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.setUrl("http://abc123456abc.efg.com");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(2000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        request.appendRequest("{}");
        EasyMock.expectLastCall().anyTimes();
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        eventListenerManager.listen(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(callback, status, response, entity, request, llmCallback, eventListenerManager);
        router.setEventListenerService(eventListenerManager);
        try {
            router.route(request, new LLMConfig(), llmCallback);
        } finally {
            EasyMock.verify(callback, status, response, entity, request, llmCallback, eventListenerManager);
            client.close();
        }
    }

    @Test
    public void testBuildHeader() throws Exception {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "https://www.w3.org/";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        HttpPost post = new HttpPost("http://1.2.3.com");
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHeaders(Collections.singletonMap("HELLO", "WORLD"));
        router.buildHeaders(llmConfig, post);
        Assert.assertEquals("WORLD", post.getFirstHeader("HELLO").getValue());
    }

    @Test(expected = IllegalArgumentException.class)
    public void testRouteEmptyUrl() throws Exception {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        ProviderRequest req = new ProviderRequest();
        req.setStream(true);
        router.route(req, new LLMConfig(), null);
    }

    @Test
    public void testBuildHeadersNull() throws Exception {
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "http://x";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        LLMConfig config = new LLMConfig();
        config.setHeaders(null);
        HttpPost post = new HttpPost("http://x");
        router.buildHeaders(config, post);
        Assert.assertEquals("application/json", post.getFirstHeader("Content-Type").getValue());
    }

    /**
     * buildTimeout：request.getTimeout() 非 null 时直接采用，不解析 upstream / FunCall
     */
    @Test
    public void testBuildTimeout_requestTimeoutTakesPrecedence() throws Exception {
        ProviderRequest request = new ProviderRequest();
        request.setTimeout(50001);
        request.setMessage(messageWithUpstream("", new HashMap<>()));
        request.setUpstreamTimeout(70001);
        request.setFunCallTimeout(80001);
        ProviderRouter<ProviderRequest> router = newStubRouter();
        router.setTimeout(90001);
        Assert.assertEquals(Integer.valueOf(50001), router.buildTimeout(request, new LLMConfig()));
    }

    /**
     * buildTimeout：timeout 为空且 upstream 非空时取 getUpstreamTimeout()
     */
    @Test
    public void testBuildTimeout_upstreamNonEmpty_returnsUpstreamTimeout() throws Exception {
        ProviderRequest request = new ProviderRequest();
        request.setMessage(messageWithUpstream("up", new HashMap<>()));
        request.setUpstreamTimeout(61000);
        ProviderRouter<ProviderRequest> router = newStubRouter();
        router.setTimeout(99999);
        Assert.assertEquals(Integer.valueOf(61000), router.buildTimeout(request, new LLMConfig()));
    }

    /**
     * buildTimeout：upstream 非空但 getUpstreamTimeout() 为 null 时回落 this.timeout
     */
    @Test
    public void testBuildTimeout_upstreamNonEmpty_upstreamTimeoutNull_fallsBackToRouterDefault() throws Exception {
        ProviderRequest request = new ProviderRequest();
        request.setMessage(messageWithUpstream("up", new HashMap<>()));
        request.setUpstreamTimeout(null);
        ProviderRouter<ProviderRequest> router = newStubRouter();
        router.setTimeout(62000);
        Assert.assertEquals(null, router.buildTimeout(request, new LLMConfig()));
    }

    /**
     * buildTimeout：timeout 为空、upstream 为空、isFromFunCall 为 true 时取 getFunCallTimeout()
     */
    @Test
    public void testBuildTimeout_funCallBranch_returnsFunCallTimeout() throws Exception {
        Map<String, Object> meta = new HashMap<>();
        meta.put(ProviderRequestService.KEY_FUN_FETCH, new Object());
        ProviderRequest request = new ProviderRequest();
        request.setMessage(messageWithUpstream("", meta));
        request.setFunCallTimeout(12345);
        ProviderRouter<ProviderRequest> router = newStubRouter();
        router.setTimeout(9999);
        Assert.assertEquals(Integer.valueOf(12345), router.buildTimeout(request, new LLMConfig()));
    }

    /**
     * buildTimeout：timeout 为空、upstream 为空、非 FunCall 时回落 this.timeout
     */
    @Test
    public void testBuildTimeout_noUpstreamNotFunCall_fallsBackToRouterDefault() throws Exception {
        ProviderRequest request = new ProviderRequest();
        request.setMessage(messageWithUpstream("", new HashMap<>()));
        ProviderRouter<ProviderRequest> router = newStubRouter();
        router.setTimeout(33333);
        Assert.assertEquals(null, router.buildTimeout(request, new LLMConfig()));
    }

    /**
     * buildTimeout：FunCall 分支上 getFunCallTimeout() 为 null 时回落 this.timeout
     */
    @Test
    public void testBuildTimeout_funCallBranch_funCallTimeoutNull_fallsBackToRouterDefault() throws Exception {
        Map<String, Object> meta = new HashMap<>();
        meta.put(ProviderRequestService.KEY_FUN_MERGE, new Object());
        ProviderRequest request = new ProviderRequest();
        request.setMessage(messageWithUpstream("", meta));
        request.setFunCallTimeout(null);
        ProviderRouter<ProviderRequest> router = newStubRouter();
        router.setTimeout(44444);
        Assert.assertEquals(null, router.buildTimeout(request, new LLMConfig()));
    }

    private static Message messageWithUpstream(String upstream, Map<String, Object> metadata) {
        LLMQuery q = ObjectBuilder.buildLLMQuery(metadata);
        NettyRequest work = (NettyRequest) ((LLMQueryDelegate) q).getWorkTask();
        work.setUpstream(upstream);
        return Message.build(q);
    }

    @Test(expected = RuntimeException.class)
    public void testBuildRouterException() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setMaxError(1000);
        req.setToken("Token");
        req.setStream(true);
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(status.getStatusCode()).andReturn(200).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{})).anyTimes();
        req.setContainHistories(true);
        req.setMessage(message);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        AtomicBoolean release = new AtomicBoolean(false);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {

            }

            @Override
            public FutureCallback<Void> consuming(Executor executor) throws Exception {
                throw new RuntimeException();
            }

            @Override
            public void released() {
                release.set(true);
            }
        };
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return reader;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "https://www.w3.org/";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        router.setStream(client);
        router.setOnce(client);
        router.setTimeoutRate(2D);
        router.setTimeout(5000);
        router.setMaxSize(1024);
        router.setMaxSizeRate(1.0);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        Assert.assertNotNull(router.getHttpClientConfig());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(null).anyTimes();
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        request.appendRequest("{}");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        eventListenerManager.listen(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(callback, status, response, request, entity, llmCallback, eventListenerManager);
        router.setEventListenerService(eventListenerManager);
        router.setExecutorService(Executors.newFixedThreadPool(1));
        try {
            router.route(request, new LLMConfig(), llmCallback);
        } finally {
            Assert.assertTrue(release.get());
            EasyMock.verify(callback, status, response, entity, request, llmCallback, eventListenerManager);
        }
    }

    /**
     * 覆盖 route() 的 catch (Exception e)：execute/consuming 抛异常时执行 WorkflowException.dolog、httpRequest.abort()、reader.released() 并重新抛出
     */
    @Test(expected = RuntimeException.class)
    public void testRoute_catchCallsDologAbortReleasedAndRethrows() throws Exception {
        ProviderRequest req = new ProviderRequest();
        req.setToken("Token");
        req.setStream(true);
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setContainHistories(true);
        req.setTokenBuffer(100);
        req.setTokenFirst(100);
        req.setHistories(10);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        AtomicBoolean released = new AtomicBoolean(false);
        ProviderReader reader = new ProviderReader(ProviderReaderConfig.<ProviderRequest>builder()
                .request(req)
                .llmCallback(callback)
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(0)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            @Override
            protected void completed(String message) throws Exception {
            }

            @Override
            public FutureCallback<Void> consuming(Executor executor) throws Exception {
                throw new RuntimeException("route fail");
            }

            @Override
            public void released() {
                released.set(true);
            }
        };
        ProviderRouter router = new ProviderRouter() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return reader;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return "https://example.com/";
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }

            @Override
            protected HttpPost buildRequest(ProviderRequest request, LLMConfig llmConfig, String httpURL) throws Exception {
                return new HttpPost(httpURL);
            }
        };
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        router.setStream(client);
        router.setOnce(client);
        router.setTimeout(5000);
        router.setTimeoutRate(2d);
        router.setMaxSize(1024);
        router.setMaxSizeRate(1.0);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.setUrl(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(null).anyTimes();
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        request.appendRequest(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(req.getMessage()).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        LLMCallback llmCallback = EasyMock.createMock(LLMCallback.class);
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        eventListenerManager.listen(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request, llmCallback, eventListenerManager);
        router.setEventListenerService(eventListenerManager);
        router.setExecutorService(Executors.newFixedThreadPool(1));
        try {
            router.route(request, new LLMConfig(), llmCallback);
        } finally {
            Assert.assertTrue("reader.released() should have been called", released.get());
            EasyMock.verify(request, llmCallback, eventListenerManager);
            client.close();
        }
    }

    private static ProviderRouter<ProviderRequest> newStubRouter() {
        return new ProviderRouter<ProviderRequest>() {
            @Override
            protected ProviderReader reader(ProviderRequest request, LLMConfig llmConfig, LLMCallback llmCallback) {
                return null;
            }

            @Override
            protected String url(ProviderRequest request, LLMConfig llmConfig, String t) {
                return null;
            }

            @Override
            protected Object body(ProviderRequest request) {
                return "{}";
            }
        };
    }
}
