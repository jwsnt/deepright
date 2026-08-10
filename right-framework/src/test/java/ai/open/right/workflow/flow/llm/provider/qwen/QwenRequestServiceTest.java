package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.impl.LLMPromptServiceImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestRewriter;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class QwenRequestServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = QwenRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = QwenRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testBuild() throws Exception {
        QwenRequestService s = new QwenRequestService();
        Assert.assertEquals(s.build().getClass(), OpenAiRequest.class);
    }

    @Test
    public void testConfig() throws Exception {
        LLMPromptServiceImpl prompt = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(prompt.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(prompt);
        QwenRequestService s = new QwenRequestService();
        s.setModel("model");
        s.setToken("token");
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        s.setLlmPromptService(prompt);
        s.setProviderToken(new ProviderToken());
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(QwenRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(QwenRequestService.KEY_TOKEN, "TOKEN_A");
        add.put(QwenRequestService.KEY_MAX_TOKENS, "1024");
        add.put(QwenRequestService.KEY_FREQUENCY_PENALTY, "0.7");
        add.put(QwenRequestService.KEY_TOP_P, "2.0");
        add.put(QwenRequestService.KEY_MODEL, "Model");
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
        QwenRequestService pro = new QwenRequestService();
        pro.setModel("model");
        pro.setToken("token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(QwenRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(QwenRequestService.KEY_TEMPERATURE, "0.3");
        add.put(QwenRequestService.KEY_MODEL, "model");
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
        pro.request(req, config, message);
        Assert.assertEquals("model", req.getModel());
        Assert.assertEquals(req.getStream(), true);
        Assert.assertEquals(req.getContainHistories(), true);
        Assert.assertEquals(req.getHistories(), Integer.valueOf(20));
        Assert.assertEquals(req.getTokenFirst(), Integer.valueOf(10));
        Assert.assertEquals(req.getTokenBuffer(), Integer.valueOf(20));
        EasyMock.verify(pservice);
    }

    @Test
    public void testGetModel() throws Exception {
        QwenRequestService service = new QwenRequestService();
        service.setModel("qwen-turbo");
        Assert.assertEquals("qwen-turbo", service.getModel(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testReasoningDoesNothingWithoutThinkingConfig() throws Exception {
        QwenRequestService service = new QwenRequestService();
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, new LLMConfig(), ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assert.assertNull(request.getExtraBody());
        Assert.assertNull(request.getReasoningEffort());
    }

    @Test
    public void testReasoningPrefersQueryThinkingAndEffort() throws Exception {
        QwenRequestService service = new QwenRequestService();
        service.setReasoningEffort("service-effort");
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "EnAbLeD"));
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "query-effort");
        LLMConfig config = new LLMConfig();
        Map<String, Object> additional = new HashMap<String, Object>();
        additional.put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        additional.put(ProviderRequestService.KEY_REASONING_EFFORT, "config-effort");
        config.setAdditional(additional);
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(Boolean.TRUE, request.getExtraBody().get("enable_thinking"));
        Assert.assertEquals("query-effort", request.getReasoningEffort());
    }

    @Test
    public void testReasoningUsesConfigAdaptiveThinkingAndEffortWhenQueryValuesAreEmpty() throws Exception {
        QwenRequestService service = new QwenRequestService();
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.emptyMap());
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "");
        LLMConfig config = new LLMConfig();
        Map<String, Object> additional = new HashMap<String, Object>();
        additional.put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "adaptive"));
        additional.put(ProviderRequestService.KEY_REASONING_EFFORT, "config-effort");
        config.setAdditional(additional);
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(Boolean.TRUE, request.getExtraBody().get("enable_thinking"));
        Assert.assertEquals("config-effort", request.getReasoningEffort());
    }

    @Test
    public void testReasoningUsesServiceEffortFallback() throws Exception {
        QwenRequestService service = new QwenRequestService();
        service.setReasoningEffort("service-effort");
        LLMConfig config = new LLMConfig();
        Map<String, Object> additional = new HashMap<String, Object>();
        additional.put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"));
        additional.put(ProviderRequestService.KEY_REASONING_EFFORT, "");
        config.setAdditional(additional);
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, config, ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assert.assertEquals(Boolean.TRUE, request.getExtraBody().get("enable_thinking"));
        Assert.assertEquals("service-effort", request.getReasoningEffort());
    }

    @Test
    public void testReasoningDisablesThinkingForUnsupportedType() throws Exception {
        QwenRequestService service = new QwenRequestService();
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        LLMConfig config = new LLMConfig();
        config.setAdditional(Collections.singletonMap(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled")));
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(Boolean.FALSE, request.getExtraBody().get("enable_thinking"));
        Assert.assertNull(request.getReasoningEffort());
    }
}
