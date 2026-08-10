package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.config.impl.LLMPromptServiceImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.apache.commons.lang3.reflect.MethodUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class DeepSeekRequestServiceTest {


    @Test
    public void testBuild() throws Exception {
        DeepSeekRequestService s = new DeepSeekRequestService();
        Assert.assertEquals(s.build().getClass(), OpenAiRequest.class);
    }

    @Test
    public void testConfig() throws Exception {
        LLMPromptServiceImpl prompt = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(prompt.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(prompt);
        DeepSeekRequestService s = new DeepSeekRequestService();
        s.setProviderToken(new ProviderToken());
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        s.setLlmPromptService(prompt);
        s.setModel("model");
        s.setToken("token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(DeepSeekRequestService.KEY_EXTRA_BODY, new HashMap<>(Collections.singletonMap("Hello", "World")));
        add.put(DeepSeekRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(DeepSeekRequestService.KEY_TOKEN, "TOKEN_A");
        add.put(DeepSeekRequestService.KEY_MAX_TOKENS, "1024");
        add.put(DeepSeekRequestService.KEY_FREQUENCY_PENALTY, "0.7");
        add.put(DeepSeekRequestService.KEY_TOP_P, "2.0");
        add.put(DeepSeekRequestService.KEY_MODEL, "Model");
        config.setAdditional(add);
        OpenAiRequest req = s.config(config, ObjectBuilder.buildLLMQuery());
        Assert.assertNotNull(req);
        EasyMock.verify(prompt);
    }

    @Test
    public void testRequest() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setMessage(message);
        req.setPrompt("Prompt");
        req.setStream(true);
        req.setHistories(10);
        req.setTokenFirst(100);
        req.setTokenBuffer(100);
        req.setContainHistories(true);
        LLMPromptServiceImpl pservice = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(pservice.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(pservice);
        DeepSeekRequestService pro = new DeepSeekRequestService();
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(DeepSeekRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(DeepSeekRequestService.KEY_TEMPERATURE, "0.3");
        add.put(DeepSeekRequestService.KEY_PRESENCE_PENALTY, "2.0");
        add.put(DeepSeekRequestService.KEY_MODEL, "model");
        config.setAdditional(add);
        config.setContainHistories(true);
        config.setTokenBuffer(20);
        config.setTokenFirst(10);
        config.setHistories(20);
        config.setStream(true);
        pro.setProviderToken(new ProviderToken());
        pro.setLlmPromptService(pservice);
        pro.setModel("model");
        pro.setToken("token");
        MethodUtils.invokeMethod(pro, true, "request", req, config, message);
        Assert.assertEquals("model", req.getModel());
        Assert.assertEquals(req.getStream(), true);
        Assert.assertEquals(Double.valueOf(2.0), req.getPresencePenalty());
        Assert.assertEquals(req.getContainHistories(), true);
        Assert.assertEquals(req.getHistories(), Integer.valueOf(20));
        Assert.assertEquals(req.getTokenFirst(), Integer.valueOf(10));
        Assert.assertEquals(req.getTokenBuffer(), Integer.valueOf(20));
        EasyMock.verify(pservice);
    }

    @Test
    public void testRequestWithFunCallAnsStream() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setFunCalls(List.of(new LLMFunCall()));
        req.setMessage(message);
        req.setPrompt("Prompt");
        req.setStream(true);
        req.setHistories(10);
        req.setTokenFirst(100);
        req.setTokenBuffer(100);
        req.setContainHistories(true);
        LLMPromptServiceImpl pservice = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(pservice.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(pservice);
        DeepSeekRequestService pro = new DeepSeekRequestService();
        pro.setModel("model");
        pro.setToken("token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(DeepSeekRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(DeepSeekRequestService.KEY_TEMPERATURE, "0.3");
        add.put(DeepSeekRequestService.KEY_PRESENCE_PENALTY, "2.0");
        add.put(DeepSeekRequestService.KEY_MODEL, "model");
        config.setAdditional(add);
        config.setContainHistories(true);
        config.setTokenBuffer(20);
        config.setTokenFirst(10);
        config.setHistories(20);
        config.setStream(true);
        pro.setProviderToken(new ProviderToken());
        pro.setLlmPromptService(pservice);
        MethodUtils.invokeMethod(pro, true, "request", req, config, message);
        Assert.assertEquals(req.getStream(), true);
        EasyMock.verify(pservice);
    }

    @Test
    public void testReasoning() throws Exception {
        DeepSeekRequestService s = new DeepSeekRequestService();
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        Map<String, Object> thinking = Collections.singletonMap("type", "enabled");
        add.put(DeepSeekRequestService.KEY_THINKING, thinking);
        config.setAdditional(add);
        OpenAiRequest req = new OpenAiRequest();
        s.reasoning(req, config, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Collections.singletonMap("thinking", thinking), req.getExtraBody());
    }

    @Test
    public void testGetModel() throws Exception {
        DeepSeekRequestService service = new DeepSeekRequestService();
        service.setModel("deepseek-chat");
        Assert.assertEquals("deepseek-chat", service.getModel(ObjectBuilder.buildWorkflowTask()));
    }
}
