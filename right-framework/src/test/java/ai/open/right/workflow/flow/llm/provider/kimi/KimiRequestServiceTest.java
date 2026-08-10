package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
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

public class KimiRequestServiceTest {

    /**
     * 仅用于调用 protected 的 {@link KimiRequestService#adapt}
     */
    private static final class KimiAdaptProbe extends KimiRequestService {
        void invokeAdapt(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            adapt(request, llmConfig, llmQuery);
        }

        void invokeReasoningEffort(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            reasoningEffort(request, llmConfig, llmQuery);
        }

        void invokeReasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            reasoning(request, llmConfig, llmQuery);
        }
    }

    @Test
    public void adapt_kimiK25Model_setsTemperatureToOne() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        OpenAiRequest request = new OpenAiRequest();
        request.setModel(KimiRequestService.MODEL);
        request.setTemperature(0.3);
        service.invokeAdapt(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Double.valueOf(1.0), request.getTemperature());
    }

    @Test
    public void adapt_kimiK25Model_caseInsensitive_setsTemperatureToOne() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("KIMI-k2.5");
        request.setTemperature(2.0);
        service.invokeAdapt(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Double.valueOf(1.0), request.getTemperature());
    }

    @Test
    public void adapt_otherModel_setsTemperatureToOne() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("moonshot-v1-8k");
        request.setTemperature(0.77);
        service.invokeAdapt(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Double.valueOf(1.0), request.getTemperature());
    }

    @Test
    public void reasoningEffort_queryValueOverridesConfigAndServiceDefault() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_REASONING_EFFORT, "config")));
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "query");

        OpenAiRequest request = new OpenAiRequest();
        service.invokeReasoningEffort(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals("query", request.getReasoningEffort());
    }

    @Test
    public void reasoningEffort_configValueUsedWhenQueryValueIsEmpty() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_REASONING_EFFORT, "config")));
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "");

        OpenAiRequest request = new OpenAiRequest();
        service.invokeReasoningEffort(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals("config", request.getReasoningEffort());
    }

    @Test
    public void reasoningEffort_serviceDefaultUsedWhenQueryAndConfigValuesAreEmpty() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_REASONING_EFFORT, "")));

        OpenAiRequest request = new OpenAiRequest();
        service.invokeReasoningEffort(request, config, ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assert.assertEquals("service", request.getReasoningEffort());
    }

    @Test
    public void reasoning_k3ModelForcesThinkingAndSetsReasoningEffort() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "query");
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"))));
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("KIMI-K3-preview");

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(KimiRequestService.THINK_CONFIG, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertEquals("query", request.getReasoningEffort());
    }

    @Test
    public void reasoning_nonK3QueryThinkingEnabledTakesPrecedenceAndSetsReasoningEffort() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        Map<String, Object> queryThinking = Collections.singletonMap("type", "enabled");
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, queryThinking);
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "query");
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"))));
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("moonshot-v1-8k");

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(queryThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertEquals("query", request.getReasoningEffort());
    }

    @Test
    public void reasoning_nonK3QueryThinkingDisabledDoesNotSetReasoningEffort() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        Map<String, Object> queryThinking = Collections.singletonMap("type", "disabled");
        Map<String, Object> metadata = new HashMap<>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, queryThinking);
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"))));
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("moonshot-v1-8k");

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(queryThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertNull(request.getReasoningEffort());
    }

    @Test
    public void reasoning_nonK3ConfigAdaptiveThinkingSetsServiceDefaultReasoningEffort() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        service.setReasoningEffort("service");
        Map<String, Object> configThinking = Collections.singletonMap("type", "adaptive");
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Collections.singletonMap(ProviderRequestService.KEY_THINKING, configThinking)));
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("moonshot-v1-8k");

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assert.assertEquals(configThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertEquals("service", request.getReasoningEffort());
    }

    @Test
    public void reasoning_nonK3WithoutThinkingConfigurationLeavesRequestUnchanged() throws Exception {
        KimiAdaptProbe service = new KimiAdaptProbe();
        OpenAiRequest request = new OpenAiRequest();
        request.setModel("moonshot-v1-8k");

        service.invokeReasoning(request, new LLMConfig(), ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assert.assertNull(request.getExtraBody());
        Assert.assertNull(request.getReasoningEffort());
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = KimiRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = KimiRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testBuild() throws Exception {
        KimiRequestService s = new KimiRequestService();
        Assert.assertEquals(s.build().getClass(), OpenAiRequest.class);
    }

    @Test
    public void testConfig() throws Exception {
        LLMPromptServiceImpl prompt = EasyMock.createMock(LLMPromptServiceImpl.class);
        EasyMock.expect(prompt.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("Hello").anyTimes();
        EasyMock.replay(prompt);
        KimiRequestService s = new KimiRequestService();
        s.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        s.setLlmPromptService(prompt);
        s.setProviderToken(new ProviderToken());
        s.setModel("model");
        s.setToken("token");
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(KimiRequestService.KEY_TOKEN, "TOKEN_A");
        config.setAdditional(add);
        OpenAiRequest req = s.config(config, ObjectBuilder.buildLLMQuery());
        Assert.assertNotNull(req);
        EasyMock.verify(prompt);
    }

    @Test
    public void testRequest() throws Exception {
        OpenAiRequest req = new OpenAiRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        req.setResponseFormat(Collections.singletonMap("Hello", "World"));
        req.setModel("Model");
        req.setToken("Token");
        req.setTopP(2.0);
        req.setFrequencyPenalty(2.0);
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
        KimiRequestService pro = new KimiRequestService();
        LLMConfig config = new LLMConfig();
        Map<String, Object> add = new HashMap<String, Object>();
        add.put(KimiRequestService.KEY_MIMETYPE, "image");
        add.put(KimiRequestService.KEY_FREQUENCY_PENALTY, 2.0D);
        add.put(KimiRequestService.KEY_TOP_P, 3.0D);
        add.put(KimiRequestService.KEY_MAX_TOKENS, 1000);
        add.put(KimiRequestService.KEY_MODEL, "model");
        add.put(KimiRequestService.KEY_RESPONSE_SCHEMA, Collections.singletonMap("Hello", "World"));
        add.put(KimiRequestService.KEY_TEMPERATURE, "0.3");
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
        Assert.assertEquals(req.getModel(), "model");
        Assert.assertEquals(req.getMaxTokens(), Integer.valueOf(1000));
        Assert.assertEquals(req.getTopP(), Double.valueOf(3.0D));
        Assert.assertNull(req.getResponseFormat().get("Hello"));
        Assert.assertEquals("Bearer token", req.getToken());
        Assert.assertEquals("model", req.getModel());
        Assert.assertEquals(req.getFrequencyPenalty(), Double.valueOf(2.0D));
        Assert.assertEquals(req.getStream(), true);
        Assert.assertEquals(req.getContainHistories(), true);
        Assert.assertEquals(req.getHistories(), Integer.valueOf(20));
        Assert.assertEquals(req.getTokenFirst(), Integer.valueOf(10));
        Assert.assertEquals(req.getTokenBuffer(), Integer.valueOf(20));
        EasyMock.verify(pservice);
    }

    @Test
    public void testGetModel() throws Exception {
        KimiRequestService service = new KimiRequestService();
        service.setModel("moonshot-v1-32k");
        Assert.assertEquals("moonshot-v1-32k", service.getModel(ObjectBuilder.buildWorkflowTask()));
    }
}
