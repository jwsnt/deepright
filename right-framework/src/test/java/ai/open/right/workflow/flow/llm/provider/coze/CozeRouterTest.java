package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.HttpPost;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class CozeRouterTest {

    @Test
    public void testInit() throws Exception {
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        CozeRequest request = EasyMock.createMock(CozeRequest.class);
        CozeRouter router = new CozeRouter();
        router.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        router.setBuffer(1024);
        router.setTimeout(5000);
        router.setQueue(1025);
        router.setDiscard(1025);
        router.setQueueTimeout(1024);
        router.setCapacity(1024);
        CozeReader cread = router.reader(request, new LLMConfig(), callback);
        Assert.assertEquals(cread.getProviderReaderCallback().getLlmCallback(), callback);
        Assert.assertEquals(cread.getRequest(), request);
    }

    @Test
    public void testReconfig() throws Exception {
        CozeRequest request = EasyMock.createMock(CozeRequest.class);
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("Hello").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getUpstreamTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        HttpPost post = EasyMock.createMock(HttpPost.class);
        post.addHeader("Authorization", "Hello");
        EasyMock.expectLastCall().anyTimes();
        post.setConfig(EasyMock.anyObject(RequestConfig.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request, post);
        CozeRouter router = new CozeRouter();
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
        router.reConfig(request, new LLMConfig(), post);
        EasyMock.verify(request, post);
    }

    @Test
    public void getURL() throws Exception {
        CozeRouter router = new CozeRouter();
        CozeRequest request = new CozeRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        router.setUrl("Url");
        Assert.assertEquals("Url", router.url(request, null, null));
    }

    @Test
    public void getBody() throws Exception {
        CozeRequest req = new CozeRequest();
        req.setBotId("BotID");
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        req.setContainHistories(false);
        req.setTokenBuffer(10);
        req.setTokenFirst(20);
        req.setHistories(10);
        req.setStream(false);
        History history = new History();
        history.setContent("Content");
        history.setRole(History.ROLE_ASSISTANT);
        history.setType(History.TYPE_ANSWER);
        message.addHistories(Arrays.asList(history));
        CozeRouter router = new CozeRouter();
        Assert.assertNotNull(router.body(req));
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = CozeRouter.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = CozeRouter.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
