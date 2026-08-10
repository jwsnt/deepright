package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRouterReflectTestUtil;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.HttpPost;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class QwenRouterTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = QwenRouter.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = QwenRouter.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testInit() throws Exception {
        OpenAiRequest request = EasyMock.createMock(OpenAiRequest.class);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        QwenRouter router = new QwenRouter();
        router.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        router.setBuffer(1024);
        router.setQueue(1025);
        router.setTimeout(1026);
        router.setDiscard(1026);
        router.setQueueTimeout(1024);
        router.setCapacity(1024);
        OpenAiReader reader = OpenAiRouterReflectTestUtil.invokeReader(router, request, new LLMConfig(), callback);
        Assert.assertEquals(reader.getProviderReaderCallback().getLlmCallback(), callback);
        Assert.assertEquals(reader.getRequest(), request);
    }

    @Test
    public void testReconfig() throws Exception {
        OpenAiRequest request = EasyMock.createMock(OpenAiRequest.class);
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("Token").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        HttpPost post = EasyMock.createMock(HttpPost.class);
        post.addHeader("Authorization", "Token");
        EasyMock.expectLastCall().anyTimes();
        post.setConfig(EasyMock.anyObject(RequestConfig.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request, post);
        QwenRouter router = new QwenRouter();
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        Assert.assertNotNull(router.getHttpClientConfig());
        router.setTimeout(1000);
        router.setTimeoutRate(2.0D);
        OpenAiRouterReflectTestUtil.invokeReConfig(router, request, new LLMConfig(), post);
        EasyMock.verify(request, post);
    }

    @Test
    public void getURL() throws Exception {
        QwenRouter router = new QwenRouter();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        router.setUrl("URL");
        Assert.assertEquals("URL", router.url(request, null, null));
    }

    @Test
    public void getBody() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setContainHistories(false);
        req.setMessage(message);
        req.setFrequencyPenalty(1.0);
        req.setTokenBuffer(10);
        req.setTokenFirst(20);
        req.setHistories(10);
        req.setStream(false);
        req.setPrompt("HELLO");
        History his = new History();
        his.setContent("Content");
        his.setRole(History.ROLE_ASSISTANT);
        his.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(his));
        QwenRouter router = new QwenRouter();
        Assert.assertNotNull(router.body(req));
    }
}
