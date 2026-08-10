package ai.open.right.workflow.flow.llm.provider.deepseek;

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
import org.apache.http.client.methods.HttpPost;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class DeepSeekRouterTest {

    @Test
    public void testInit() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMCallback callback = m -> {
        };
        DeepSeekRouter router = new DeepSeekRouter();
        router.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        router.setBuffer(1024);
        router.setTimeout(1025);
        router.setQueue(1026);
        router.setDiscard(1026);
        router.setQueueTimeout(1024);
        router.setCapacity(1024);
        OpenAiReader reader = OpenAiRouterReflectTestUtil.invokeReader(router, request, new LLMConfig(), callback);
        Assert.assertEquals(reader.getProviderReaderCallback().getLlmCallback(), callback);
        Assert.assertEquals(reader.getRequest(), request);
    }

    @Test
    public void testReconfig() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setToken("Token");
        request.setStream(true);
        request.setUpstreamTimeout(1000);
        request.setFunCallTimeout(1000);
        HttpPost post = new HttpPost("http://localhost/");
        DeepSeekRouter router = new DeepSeekRouter();
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
        Assert.assertEquals("Token", post.getFirstHeader("Authorization").getValue());
        Assert.assertNotNull(post.getConfig());
    }

    @Test
    public void getURL() throws Exception {
        DeepSeekRouter router = new DeepSeekRouter();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        router.setUrl("URL");
        Assert.assertEquals("URL", router.url(request, null, null));
    }

    @Test
    public void getURL_requestUrlOverridesBeanDefault() throws Exception {
        DeepSeekRouter router = new DeepSeekRouter();
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setUrl("https://deepseek.override/v1");
        router.setUrl("URL");
        Assert.assertEquals("https://deepseek.override/v1", router.url(request, null, null));
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
        DeepSeekRouter router = new DeepSeekRouter();
        Assert.assertNotNull(router.body(req));
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = DeepSeekRouter.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = DeepSeekRouter.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testDeepSeekMessageNullThinking() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setExtraBody(null);
        Object msg = DeepSeekRouterReflectTestUtil.newOpenAiMessage(req);
        Assert.assertNull(DeepSeekRouterReflectTestUtil.getThinking(msg));
    }

    @Test
    public void testDeepSeekContentNullReasoning() throws Exception {
        History h = new History();
        h.setRole(History.ROLE_USER);
        h.setContent("C");
        h.setReason(null);
        Object content = DeepSeekRouterReflectTestUtil.newOpenAiContent(h);
        Assert.assertNull(DeepSeekRouterReflectTestUtil.getReasoning(content));
    }
}
