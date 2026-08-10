package ai.open.right.workflow.flow.llm.provider.volcengine;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.Assert;
import org.junit.Test;

import java.lang.reflect.Method;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class VolcengineRequestServiceTest {

    private static final class VolcengineReasoningProbe extends VolcengineRequestService {
        void invokeReasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            reasoning(request, llmConfig, llmQuery);
        }
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = VolcengineRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = VolcengineRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        VolcengineRequestService volcengineRequestService = new VolcengineRequestService() {
            protected void buildPrompt(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            }

            // 从历史记录恢复消息
            protected void internalHistory(OpenAiRequest request) throws Exception {

            }
        };
        volcengineRequestService.setToken("Hello World");
        volcengineRequestService.setProviderToken(new ProviderToken());
        OpenAiRequest openAiRequest = new OpenAiRequest();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(Collections.singletonMap("token", "HELLO"));
        Method requestMethod = VolcengineRequestService.class.getSuperclass().getDeclaredMethod("request", OpenAiRequest.class, LLMConfig.class, ai.open.right.workflow.flow.llm.LLMQuery.class);
        requestMethod.setAccessible(true);
        requestMethod.invoke(volcengineRequestService, openAiRequest, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals("doubao-seed-2-1-turbo-260628", openAiRequest.getModel());
        Assert.assertEquals("Bearer HELLO", openAiRequest.getToken());
    }

    @Test
    public void testWithModel() throws Exception {
        VolcengineRequestService volcengineRequestService = new VolcengineRequestService() {
            protected void buildPrompt(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            }

            // 从历史记录恢复消息
            protected void internalHistory(OpenAiRequest request) throws Exception {

            }
        };
        volcengineRequestService.setToken("Hello World");
        volcengineRequestService.setProviderToken(new ProviderToken());
        OpenAiRequest openAiRequest = new OpenAiRequest();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(Collections.singletonMap("model", "HELLO"));
        Method requestMethod = VolcengineRequestService.class.getSuperclass().getDeclaredMethod("request", OpenAiRequest.class, LLMConfig.class, ai.open.right.workflow.flow.llm.LLMQuery.class);
        requestMethod.setAccessible(true);
        requestMethod.invoke(volcengineRequestService, openAiRequest, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals("HELLO", openAiRequest.getModel());
        Assert.assertEquals("Bearer Hello World", openAiRequest.getToken());
    }

    @Test
    public void testGetModel() throws Exception {
        VolcengineRequestService service = new VolcengineRequestService();
        service.setModel("doubao-pro-32k");
        Assert.assertEquals("doubao-pro-32k", service.getModel(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void reasoningPrefersQueryThinkingAndEffort() throws Exception {
        Map<String, Object> queryThinking = new HashMap<String, Object>();
        queryThinking.put("type", "EnAbLeD");
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, queryThinking);
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "high");

        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        config.getAdditional().put(ProviderRequestService.KEY_REASONING_EFFORT, "medium");
        VolcengineReasoningProbe service = new VolcengineReasoningProbe();
        service.setReasoningEffort("low");
        OpenAiRequest request = new OpenAiRequest();

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertSame(queryThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertEquals("high", request.getReasoningEffort());
    }

    @Test
    public void reasoningFallsBackToAdaptiveConfigThinkingAndConfigEffort() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.emptyMap());
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "");
        Map<String, Object> configThinking = new HashMap<String, Object>();
        configThinking.put("type", "AdApTiVe");
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_THINKING, configThinking);
        config.getAdditional().put(ProviderRequestService.KEY_REASONING_EFFORT, "medium");
        VolcengineReasoningProbe service = new VolcengineReasoningProbe();
        service.setReasoningEffort("low");
        OpenAiRequest request = new OpenAiRequest();

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertSame(configThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertEquals("medium", request.getReasoningEffort());
    }

    @Test
    public void reasoningUsesServiceEffortWhenQueryAndConfigEffortsAreEmpty() throws Exception {
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"));
        config.getAdditional().put(ProviderRequestService.KEY_REASONING_EFFORT, "");
        VolcengineReasoningProbe service = new VolcengineReasoningProbe();
        service.setReasoningEffort("low");
        OpenAiRequest request = new OpenAiRequest();

        service.invokeReasoning(request, config, ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assert.assertEquals(Collections.singletonMap("type", "enabled"), request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertEquals("low", request.getReasoningEffort());
    }

    @Test
    public void reasoningPreservesDisabledThinkingWithoutSettingEffort() throws Exception {
        Map<String, Object> disabledThinking = Collections.singletonMap("type", "disabled");
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, disabledThinking);
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "high");
        VolcengineReasoningProbe service = new VolcengineReasoningProbe();
        service.setReasoningEffort("low");
        OpenAiRequest request = new OpenAiRequest();

        service.invokeReasoning(request, new LLMConfig(), ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertSame(disabledThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
        Assert.assertNull(request.getReasoningEffort());
    }

    @Test
    public void reasoningDoesNotSetThinkingWhenQueryAndConfigAreEmpty() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.emptyMap());
        OpenAiRequest request = new OpenAiRequest();

        new VolcengineReasoningProbe().invokeReasoning(request, new LLMConfig(), ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertNull(request.getExtraBody());
        Assert.assertNull(request.getReasoningEffort());
    }
}
