package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRouter;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.apache.http.client.methods.HttpPost;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.Arrays;

public class GeminiRouterTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = GeminiRouter.class.getConstructor((Class<?>[]) null).newInstance((Object[]) null);
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = GeminiRouter.InitConfig.class.getConstructor((Class<?>[]) null).newInstance((Object[]) null);
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testInit() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("Token").anyTimes();
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        EasyMock.replay(request, callback);
        GeminiRouter router = new GeminiRouter();
        router.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        router.setBuffer(1024);
        router.setDiscard(1024);
        router.setTimeout(1025);
        router.setQueue(1026);
        router.setQueueTimeout(1024);
        router.setCapacity(1024);
        GoogleReader googleReader = router.reader(request, new LLMConfig(), callback);
        Assertions.assertEquals(googleReader.getProviderReaderCallback().getLlmCallback(), callback);
        Assertions.assertEquals(googleReader.getRequest(), request);
        EasyMock.verify(request);
    }

    @Test
    public void getURL() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("").anyTimes();
        EasyMock.expect(request.getModel()).andReturn("model").anyTimes();
        EasyMock.expect(request.getToken()).andReturn("_").anyTimes();
        request.setModel(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        LLMConfig llmConfig = new LLMConfig();
        GeminiRouter router = new GeminiRouter();
        router.setUrlStream("STREAM_#model");
        router.setUrlOnce("ONCE_#model");
        Assertions.assertEquals("STREAM_model", router.url(request, llmConfig, ProviderRouter.URL_STREAM));
        Assertions.assertEquals("ONCE_model", router.url(request, llmConfig, ProviderRouter.URL_ONCE));

        // With custom model in config
        llmConfig.getAdditional().put(GeminiRouter.MODEL, "custom-model");
        Assertions.assertEquals("STREAM_model", router.url(request, llmConfig, ProviderRouter.URL_STREAM));
        Assertions.assertEquals("ONCE_model", router.url(request, llmConfig, ProviderRouter.URL_ONCE));

        EasyMock.verify(request);
    }

    /**
     * GoogleRequest.url 非空时覆盖模板替换结果
     */
    @Test
    public void getURL_customRequestUrlOverridesTemplate() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        request.setModel("gemini-model");
        request.setUrl("https://override.example/full");
        LLMConfig llmConfig = new LLMConfig();
        GeminiRouter router = new GeminiRouter();
        router.setUrlStream("STREAM_#model");
        router.setUrlOnce("ONCE_#model");
        Assertions.assertEquals("https://override.example/full", router.url(request, llmConfig, ProviderRouter.URL_STREAM));
        Assertions.assertEquals("https://override.example/full", router.url(request, llmConfig, ProviderRouter.URL_ONCE));
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
        GeminiRouter router = new GeminiRouter();
        Assertions.assertNotNull(router.body(req));
    }

    @Test
    public void getReConfig() throws Exception {
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("TOKEN_VAL").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(request);
        LLMConfig llmConfig = new LLMConfig();
        GeminiRouter router = new GeminiRouter();
        router.setTimeoutRate(2.0D);
        router.setTimeout(1000);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        Assertions.assertNotNull(router.getHttpClientConfig());
        HttpPost post = new HttpPost("http://test");
        router.reConfig(request, llmConfig, post);
        Assertions.assertEquals("TOKEN_VAL", post.getFirstHeader("x-goog-api-key").getValue());
        EasyMock.verify(request);
    }

    @Test
    public void testInitConfig() throws Exception {
        GeminiRouter.InitConfig initConfig = new GeminiRouter.InitConfig();
        initConfig.setUrlOnce("once/#model");
        initConfig.setUrlStream("stream/#model");
        GeminiRouter router = initConfig.geminiRouter();
        Assertions.assertEquals("once/#model", router.getUrlOnce());
        Assertions.assertEquals("stream/#model", router.getUrlStream());
    }
}
