package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.impl.LLMPromptServiceImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class BigModelRequestServiceTest {


    @Test
    public void testBuild() throws Exception {
        BigModelRequestService s = new BigModelRequestService();
        Assert.assertEquals(s.build().getClass(), OpenAiRequest.class);
    }

    @Test
    public void testConfig() throws Exception {
        LLMPromptServiceImpl prompt = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(prompt.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(prompt);
        BigModelRequestService s = new BigModelRequestService();
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        s.setProviderToken(new ProviderToken());
        s.setLlmPromptService(prompt);
        s.setModel("model");
        s.setToken("token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(BigModelRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(BigModelRequestService.KEY_TOKEN, "TOKEN_A");
        add.put(BigModelRequestService.KEY_MAX_TOKENS, "1024");
        add.put(BigModelRequestService.KEY_FREQUENCY_PENALTY, "0.7");
        add.put(BigModelRequestService.KEY_TOP_P, "2.0");
        add.put(BigModelRequestService.KEY_MODEL, "Model");
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
        BigModelRequestService pro = new BigModelRequestService();
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(BigModelRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(BigModelRequestService.KEY_TEMPERATURE, "0.3");
        add.put(BigModelRequestService.KEY_PRESENCE_PENALTY, "2.0");
        add.put(BigModelRequestService.KEY_MODEL, "model");
        config.setAdditional(add);
        config.setContainHistories(true);
        config.setTokenBuffer(20);
        config.setTokenFirst(10);
        config.setHistories(20);
        config.setStream(true);
        pro.setModel("model");
        pro.setToken("token");
        pro.setProviderToken(new ProviderToken());
        pro.setLlmPromptService(pservice);
        pro.request(req, config, message);
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
    public void testGetModel() throws Exception {
        BigModelRequestService service = new BigModelRequestService();
        service.setModel("glm-4-flash");
        Assert.assertEquals("glm-4-flash", service.getModel(ObjectBuilder.buildWorkflowTask()));
    }
}
