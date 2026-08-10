package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.store.history.History;
import com.google.common.collect.ImmutableMap;
import org.apache.http.client.methods.HttpPost;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class OpenAiRouterTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = OpenAiRouter.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = OpenAiRouter.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> addition = new HashMap<String, Object>();
        addition.put(OpenAiRouter.KEY_URL, "Hello World");
        llmConfig.setAdditional(addition);
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        OpenAiRouter router = new OpenAiRouter();
        Assert.assertEquals("Hello World", router.url(request, llmConfig, ""));
    }

    /**
     * ProviderRequest.url 优先于 llmConfig.additional 与 router 默认 url
     */
    @Test
    public void testUrl_requestUrlOverridesAdditionalAndBeanDefault() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> addition = new HashMap<>();
        addition.put(OpenAiRouter.KEY_URL, "from-additional");
        llmConfig.setAdditional(addition);
        OpenAiRequest req = new OpenAiRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setUrl("https://custom.endpoint/v1");
        OpenAiRouter router = new OpenAiRouter();
        router.setUrl("bean-default-url");
        Assert.assertEquals("https://custom.endpoint/v1", router.url(req, llmConfig, ""));
    }

    @Test
    public void testReConfigHeaders() throws Exception {
        OpenAiRouter router = new OpenAiRouter();
        OpenAiRequest req = new OpenAiRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setToken("TOKEN");
        req.setStream(true);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        HttpPost post = new HttpPost("http://x");
        router.setTimeout(100);
        router.setTimeoutRate(2.0D);
        router.setHttpClientConfig(httpClientConfig);
        router.reConfig(req, new LLMConfig(), post);
        Assert.assertEquals("TOKEN", post.getFirstHeader("Authorization").getValue());
    }

    @Test
    public void testOpenAiMessageWithHistory() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message msg = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        req.setMessage(msg);
        History h = new History();
        h.setRole(History.ROLE_USER);
        h.setContent("H");
        msg.addHistory(h);
        req.setMessage(msg);
        OpenAiRouter.OpenAiMessage oam = new OpenAiRouter.OpenAiMessage(req);
        Assert.assertTrue(oam.getMessages().size() > 1);
    }

    @Test
    public void testOpenAiTool_propertiesNull_skipsParameters() throws Exception {
        LLMFunCall funCall = new LLMFunCall();
        funCall.setName("fn");
        funCall.setDescription("desc");
        funCall.setProperties(null);
        OpenAiRouter.OpenAiTool tool = new OpenAiRouter.OpenAiTool(funCall);
        Assert.assertEquals("function", tool.getType());
        Assert.assertEquals("fn", tool.getFunction().get("name"));
        Assert.assertEquals("desc", tool.getFunction().get("description"));
        Assert.assertFalse(tool.getFunction().containsKey("parameters"));
    }

    @Test
    public void testOpenAiTool_propertiesEmpty_skipsParameters() throws Exception {
        LLMFunCall funCall = new LLMFunCall();
        funCall.setName("fn");
        funCall.setDescription("desc");
        funCall.setProperties(new HashMap<>());
        OpenAiRouter.OpenAiTool tool = new OpenAiRouter.OpenAiTool(funCall);
        Assert.assertFalse(tool.getFunction().containsKey("parameters"));
    }

    @Test
    public void testOpenAiTool_propertiesNonEmpty_addsParametersObject() throws Exception {
        LLMFunCall funCall = new LLMFunCall();
        funCall.setName("weather");
        funCall.setDescription("get weather");
        Map<String, Object> props = new HashMap<>();
        props.put("city", ImmutableMap.of("type", "string"));
        funCall.setProperties(props);
        OpenAiRouter.OpenAiTool tool = new OpenAiRouter.OpenAiTool(funCall);
        @SuppressWarnings("unchecked")
        Map<String, Object> parameters = (Map<String, Object>) tool.getFunction().get("parameters");
        Assert.assertNotNull(parameters);
        Assert.assertEquals("object", parameters.get("type"));
        Assert.assertEquals(props, parameters.get("properties"));
    }
}
