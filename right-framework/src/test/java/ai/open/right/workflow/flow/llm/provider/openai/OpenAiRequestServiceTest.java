package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class OpenAiRequestServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = OpenAiRequestService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = OpenAiRequestService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testJson() throws Exception {
        OpenAiRequestService openAiRequestService = new OpenAiRequestService();
        OpenAiRequest openAiRequest = new OpenAiRequest();
        Map<String, Object> jsonschema = new HashMap<String, Object>();
        jsonschema.put("A", "B");
        openAiRequest.setResponseFormat(jsonschema);
        openAiRequestService.json(openAiRequest, null, null);
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(openAiRequest.getResponseFormat().size()));
        Assert.assertEquals("json_object", openAiRequest.getResponseFormat().get("type"));
    }

    @Test
    public void testJsonWithNull() throws Exception {
        OpenAiRequestService openAiRequestService = new OpenAiRequestService();
        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequestService.json(openAiRequest, null, null);
        Assert.assertNull(openAiRequest.getResponseFormat());
    }

    @Test
    public void testGetModel() throws Exception {
        OpenAiRequestService service = new OpenAiRequestService();
        service.setModel("gpt-4o");
        Assert.assertEquals("gpt-4o", service.getModel(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testDefTokenAndDefModel() throws Exception {
        OpenAiRequestService service = new OpenAiRequestService();
        service.setToken("TOKEN_X");
        service.setModel("gpt-4.1-mini");
        Assert.assertEquals("TOKEN_X", service.defToken(ObjectBuilder.buildWorkflowTask()));
        Assert.assertEquals("gpt-4.1-mini", service.defModel(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testReasoningUsesQueryEffortOverConfigEffort() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "high");
        LLMQuery query = ObjectBuilder.buildLLMQuery(metadata);
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_REASONING_EFFORT, "low");

        OpenAiRequest request = new OpenAiRequest();
        new OpenAiRequestService().reasoning(request, config, query);

        Assert.assertEquals(Collections.singletonMap("effort", "high"), request.getExtraBody().get(OpenAiRequestService.KEY_REASONING));
    }

    @Test
    public void testReasoningUsesConfigEffortWhenQueryEffortIsEmpty() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, "");
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_REASONING_EFFORT, "medium");

        OpenAiRequest request = new OpenAiRequest();
        new OpenAiRequestService().reasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(Collections.singletonMap("effort", "medium"), request.getExtraBody().get(OpenAiRequestService.KEY_REASONING));
    }

    @Test
    public void testReasoningFallsBackToQueryThinking() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "EnAbLeD"));
        OpenAiRequestService service = new OpenAiRequestService();
        service.setReasoningEffort("low");
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, new LLMConfig(), ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(Collections.singletonMap("effort", "low"), request.getExtraBody().get(OpenAiRequestService.KEY_REASONING));
    }

    @Test
    public void testReasoningFallsBackToConfigThinking() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"));
        OpenAiRequestService service = new OpenAiRequestService();
        service.setReasoningEffort("medium");
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertEquals(Collections.singletonMap("effort", "medium"), request.getExtraBody().get(OpenAiRequestService.KEY_REASONING));
    }

    @Test
    public void testReasoningDoesNotSetExtraWhenNoEffortAndThinkingDisabled() throws Exception {
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        LLMConfig config = new LLMConfig();
        config.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        OpenAiRequestService service = new OpenAiRequestService();
        service.setReasoningEffort("low");
        OpenAiRequest request = new OpenAiRequest();

        service.reasoning(request, config, ObjectBuilder.buildLLMQuery(metadata));

        Assert.assertNull(request.getExtraBody());
    }

    @Test
    public void testExtraEnablesParallelToolForFunctionCalls() throws Exception {
        OpenAiRequestService service = new OpenAiRequestService();
        service.setParallelTool(true);
        OpenAiRequest request = new OpenAiRequest();
        request.setFunCalls(List.of(new LLMFunCall()));

        service.extra(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());

        Assert.assertEquals(Boolean.TRUE, request.getExtraBody().get(OpenAiRequestService.KEY_PARALLEL_TOOL));
    }

    @Test
    public void testExtraDoesNotEnableParallelToolWithoutFunctionCalls() throws Exception {
        OpenAiRequestService service = new OpenAiRequestService();
        service.setParallelTool(true);
        OpenAiRequest request = new OpenAiRequest();

        service.extra(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());

        Assert.assertNull(request.getExtraBody());
    }

    @Test
    public void testExtraDoesNotEnableParallelToolWhenDisabled() throws Exception {
        OpenAiRequestService service = new OpenAiRequestService();
        service.setParallelTool(false);
        OpenAiRequest request = new OpenAiRequest();
        request.setFunCalls(List.of(new LLMFunCall()));

        service.extra(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());

        Assert.assertNull(request.getExtraBody());
    }
}
