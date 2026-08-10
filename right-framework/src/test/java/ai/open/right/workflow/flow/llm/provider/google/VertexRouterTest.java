package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.HttpPost;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.Arrays;

public class VertexRouterTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = VertexRouter.class.getConstructor((Class<?>[]) null).newInstance((Object[]) null);
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = VertexRouter.InitConfig.class.getConstructor((Class<?>[]) null).newInstance((Object[]) null);
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testInit() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        VertexRouter router = new VertexRouter();
        router.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        router.setBuffer(1024);
        router.setDiscard(1024);
        router.setTimeout(1025);
        router.setQueue(1026);
        router.setQueueTimeout(1024);
        router.setCapacity(1024);
        GoogleReader reader = router.reader(request, new LLMConfig(), callback);
        Assertions.assertEquals(reader.getProviderReaderCallback().getLlmCallback(), callback);
        Assertions.assertEquals(reader.getRequest(), request);
    }

    @Test
    public void testReconfig() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(1000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("Token").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        HttpPost post = EasyMock.createMock(HttpPost.class);
        post.addHeader("Authorization", "Token");
        EasyMock.expectLastCall().anyTimes();
        post.setConfig(EasyMock.anyObject(RequestConfig.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request, post);
        VertexRouter router = new VertexRouter();
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        Assertions.assertNotNull(router.getHttpClientConfig());
        router.setTimeout(1000);
        router.setTimeoutRate(2.0D);
        router.reConfig(request, new LLMConfig(), post);
        EasyMock.verify(request, post);
    }

    @Test
    public void getURL() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("").anyTimes();
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        EasyMock.expect(request.getToken()).andReturn("_").anyTimes();
        request.setModel("Model");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        VertexRouter router = new VertexRouter();
        router.setUrlStream("Stream#projectid#location");
        router.setUrlOnce("Once#projectid#location");
        router.setLocation("Location");
        router.setProjectId("Project");
        LLMConfig config = new LLMConfig();
        Assertions.assertEquals("StreamProjectLocation", router.url(request, config, ProviderRouter.URL_STREAM));
        Assertions.assertEquals("OnceProjectLocation", router.url(request, config, ProviderRouter.URL_ONCE));
        EasyMock.verify(request);
    }

    /** GoogleRequest.url 非空时覆盖 Vertex 模板拼接结果 */
    @Test
    public void getURL_customRequestUrlOverridesTemplate() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        request.setModel("mymodel");
        request.setUrl("https://vertex.override/custom");
        VertexRouter router = new VertexRouter();
        router.setUrlStream("Stream#projectid#location");
        router.setUrlOnce("Once#projectid#location");
        router.setLocation("Location");
        router.setProjectId("Project");
        LLMConfig config = new LLMConfig();
        Assertions.assertEquals("https://vertex.override/custom", router.url(request, config, ProviderRouter.URL_STREAM));
        Assertions.assertEquals("https://vertex.override/custom", router.url(request, config, ProviderRouter.URL_ONCE));
    }

    @Test
    public void getBody() throws Exception {
        GoogleRequest req = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setContainHistories(false);
        req.setMessage(message);
        req.setTokenBuffer(10);
        req.setTokenFirst(20);
        req.setHistories(10);
        req.setStream(false);
        History his = new History();
        his.setContent("Content");
        his.setRole(History.ROLE_ASSISTANT);
        his.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(his));
        VertexRouter router = new VertexRouter();
        Assertions.assertNotNull(router.body(req));
    }

    @Test
    public void testUrlReplacementFull() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("").anyTimes();
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        request.setModel("m1");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        VertexRouter router = new VertexRouter();
        router.setUrlStream("http://test/#projectid/#location/#model");
        router.setUrlOnce("http://test/once/#projectid/#location/#model");
        router.setProjectId("p1");
        router.setLocation("l1");
        LLMConfig config = new LLMConfig();
        Assertions.assertEquals("http://test/p1/l1/model", router.url(request, config, ProviderRouter.URL_STREAM));
        Assertions.assertEquals("http://test/once/p1/l1/model", router.url(request, config, ProviderRouter.URL_ONCE));
    }

    @Test
    public void testUrlWithConfigOverrides() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("https://www.google.com").anyTimes();
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        request.setModel("m2");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        VertexRouter router = new VertexRouter();
        router.setUrlStream("http://test/#projectid/#location/#model");
        router.setProjectId("p1");
        router.setLocation("l1");
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(VertexRouter.PROJECT_ID, "p2");
        config.getAdditional().put(VertexRouter.LOCATION, "l2");
        config.getAdditional().put(VertexRouter.MODEL, "m2");
        Assertions.assertEquals("https://www.google.com", router.url(request, config, ProviderRouter.URL_STREAM));
    }

    @Test
    public void testUrlValidationFailure() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        EasyMock.replay(request);
        VertexRouter router = new VertexRouter();
        router.setUrlStream("http://test/#projectid/#location/#model");
        router.setUrlOnce("http://test/once/#projectid/#location/#model");
        router.setProjectId(""); // Trigger Assert.hasText
        Assertions.assertThrows(IllegalArgumentException.class, () -> {
            router.url(request, new LLMConfig(), ProviderRouter.URL_STREAM);
        });
        EasyMock.verify(request);
    }

    @Test
    public void testReaderWithImageBuffer() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        VertexRouter router = new VertexRouter();
        router.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        router.setBuffer(1024);
        router.setDiscard(1024);
        router.setTimeout(1025);
        router.setQueue(100);
        router.setQueueTimeout(1024);
        router.setCapacity(1024);
        LLMConfig config = new LLMConfig();
        config.setNetworkBuffer(2048);
        GoogleReader reader = router.reader(request, config, callback);
        Assertions.assertNotNull(reader);
    }

    @Test
    public void testInitConfigBeanCreation() throws Exception {
        VertexRouter.InitConfig initConfig = new VertexRouter.InitConfig();
        initConfig.setUrlStream("stream-url");
        initConfig.setUrlOnce("once-url");
        initConfig.setProjectId("project-id");
        initConfig.setLocation("location");
        VertexRouter router = initConfig.vertexRouter();
        Assertions.assertEquals("stream-url", router.getUrlStream());
        Assertions.assertEquals("once-url", router.getUrlOnce());
        Assertions.assertEquals("project-id", router.getProjectId());
        Assertions.assertEquals("location", router.getLocation());
    }
}
