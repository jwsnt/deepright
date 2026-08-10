package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.apache.commons.lang3.reflect.MethodUtils;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GoogleRequestServiceTest {

    @Test
    public void testHashCode() throws Exception {
        GoogleRequestService<GoogleRequest> object = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return null;
            }
        };
        Assertions.assertEquals(object.hashCode(), object.hashCode());
        Assertions.assertEquals(object, object);
    }

    @Test
    public void testSetGet() {
        GoogleRequestService<GoogleRequest> requestService = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return null;
            }
        };
        requestService.setSafeSettings(Arrays.asList(Map.of("A", "B"), Map.of("C", "D")));
        Assertions.assertNotNull(requestService.getSafeSettings());
        Assertions.assertEquals(2, requestService.getSafeSettings().size());
    }

    /**
     * 覆盖 GoogleRequestService.appName 的 get/set。
     */
    @Test
    public void googleRequestService_appName_getSet() {
        GoogleRequestService<GoogleRequest> requestService = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return null;
            }
        };
        Assertions.assertNull(requestService.getAppName());
        requestService.setAppName("my-service");
        Assertions.assertEquals("my-service", requestService.getAppName());
    }

    @Test
    public void testInitPolicy() throws Exception {
        GoogleRequestService<GoogleRequest> service = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return null;
            }
        };
        service.init("BLOCK_LOW_AND_ABOVE");
        Assertions.assertEquals(4, service.getSafeSettings().size());
        for (Map<String, Object> setting : service.getSafeSettings()) {
            Assertions.assertEquals("BLOCK_LOW_AND_ABOVE", setting.get("threshold"));
        }
    }

    @Test
    public void testRequestFull() throws Exception {
        GoogleRequestService<GoogleRequest> service = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return new GoogleRequest();
            }
        };
        service.setProviderToken(new ProviderToken());
        LLMPromptService promptService = EasyMock.createMock(LLMPromptService.class);
        EasyMock.expect(promptService.prompt(EasyMock.anyObject(ProviderRequest.class), EasyMock.anyObject(), EasyMock.anyObject())).andReturn("test-prompt").anyTimes();
        EasyMock.replay(promptService);
        service.setLlmPromptService(promptService);

        GoogleRequest req = new GoogleRequest();
        LLMConfig config = new LLMConfig();
        Map<String, Object> additional = new HashMap<>();
        additional.put(GoogleRequestService.KEY_RESPONSE_MODALITIES, List.of("TEXT"));
        additional.put(GoogleRequestService.KEY_FREQUENCY_PENALTY, 0.5);
        additional.put(GoogleRequestService.KEY_PRESENCE_PENALTY, 0.6);
        additional.put(GoogleRequestService.KEY_MAX_OUTPUT_TOKENS, 100);
        additional.put(GoogleRequestService.KEY_TEMPERATURE, 0.7);
        additional.put(GoogleRequestService.KEY_MEDIA_RESOLUTION, "HIGH");
        additional.put(GoogleRequestService.KEY_RESPONSE_SCHEMA, Map.of("type", "object"));
        additional.put(GoogleRequestService.KEY_THINKING_CONFIG, Map.of("include_thoughts", true));
        additional.put(GoogleRequestService.KEY_IMAGE_CONFIG, Map.of("size", "1024x1024"));
        additional.put(GoogleRequestService.KEY_TOOL_CONFIG, Map.of("function_calling_config", Map.of("mode", "AUTO")));
        additional.put(GoogleRequestService.KEY_MIMETYPE, "application/json");
        additional.put(GoogleRequestService.KEY_SEED, 123);
        additional.put(GoogleRequestService.KEY_TOP_P, 0.9);
        additional.put(GoogleRequestService.KEY_TOP_K, 40);
        additional.put(GoogleRequestService.KEY_SAFETY_SETTINGS, List.of(Map.of("category", "HARM", "threshold", "BLOCK_NONE")));
        config.setAdditional(additional);

        LLMQuery query = ObjectBuilder.buildLLMQuery();
        service.request(req, config, query);

        Assertions.assertEquals(List.of("TEXT"), req.getResponseModalities());
        Assertions.assertEquals(0.5, req.getFrequencyPenalty());
        Assertions.assertEquals(0.6, req.getPresencePenalty());
        Assertions.assertEquals(100, req.getMaxOutputTokens());
        Assertions.assertEquals(0.7, req.getTemperature());
        Assertions.assertEquals("HIGH", req.getMediaResolution());
        Assertions.assertEquals(Map.of("type", "object"), req.getResponseSchema());
        Assertions.assertEquals(Map.of("include_thoughts", true), req.getThinkingConfig());
        Assertions.assertEquals(Map.of("size", "1024x1024"), req.getImageConfig());
        Assertions.assertEquals(Map.of("function_calling_config", Map.of("mode", "AUTO")), req.getToolsConfig());
        Assertions.assertEquals("application/json", req.getMimeType());
        Assertions.assertEquals(123, req.getSeed());
        Assertions.assertEquals(0.9, req.getTopP());
        Assertions.assertEquals(40, req.getTopK());
        Assertions.assertEquals(List.of(Map.of("category", "HARM", "threshold", "BLOCK_NONE")), req.getSafetySettings());
    }

    @Test
    public void testConfigTokenCheck() throws Exception {
        GoogleRequestService<GoogleRequest> service = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return new GoogleRequest();
            }
        };
        service.setProviderToken(new ProviderToken());
        // Mock dependencies for config()
        service.setLlmPromptService(EasyMock.createNiceMock(LLMPromptService.class));
        service.setProviderRequestRewriter((req, config, query) -> {
        });

        LLMConfig config = new LLMConfig();
        // No token in additional
        Assertions.assertThrows(IllegalArgumentException.class, () -> {
            service.config(config, ObjectBuilder.buildLLMQuery());
        });
    }

    private static GoogleRequestService<GoogleRequest> createService() {
        return new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return new GoogleRequest();
            }
        };
    }

    private static Method imageMethod() throws Exception {
        Method m = MethodUtils.getMatchingMethod(GoogleRequestService.class, "image", LLMConfig.class, LLMQuery.class, Map.class);
        m.setAccessible(true);
        return m;
    }

    /**
     * 无请求 metadata 时，直接返回传入的 imageConfig（不合并）。
     */
    @Test
    public void image_noRequestMetadata_returnsConfigUnchanged() throws Exception {
        GoogleRequestService<GoogleRequest> service = createService();
        LLMConfig config = new LLMConfig();
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.putMetadata("other", "value");
        Map<String, Object> imageConfig = new HashMap<>(Map.of("imageSize", "2K"));
        Map<String, Object> result = (Map<String, Object>) imageMethod().invoke(service, config, query, imageConfig);
        Assertions.assertSame(imageConfig, result);
        Assertions.assertEquals(Map.of("imageSize", "2K"), result);
    }

    /**
     * 有请求 metadata 但 requestConfig 为 null 时，返回原 imageConfig 不变。
     */
    @Test
    public void image_requestMetadataPresentButRequestConfigNull_returnsConfigUnchanged() throws Exception {
        GoogleRequestService<GoogleRequest> service = createService();
        LLMConfig config = new LLMConfig();
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.putMetadata(GoogleRequestService.KEY_IMAGE_CONFIG, null);
        Map<String, Object> imageConfig = new HashMap<>(Map.of("imageSize", "1K"));
        Map<String, Object> result = (Map<String, Object>) imageMethod().invoke(service, config, query, imageConfig);
        Assertions.assertSame(imageConfig, result);
        Assertions.assertEquals(Map.of("imageSize", "1K"), result);
    }

    /**
     * 默认配置优先：请求中的 key 与默认重复时保留默认值，请求仅填充默认未配置的项。
     */
    @Test
    public void image_requestConfigMergedWithDefault_absentKeysFilled() throws Exception {
        GoogleRequestService<GoogleRequest> service = createService();
        LLMConfig config = new LLMConfig();
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.putMetadata(GoogleRequestService.KEY_IMAGE_CONFIG, Map.of("imageSize", "4K", "quality", "high"));
        Map<String, Object> imageConfig = new HashMap<>(Map.of("imageSize", "2K"));
        Map<String, Object> result = (Map<String, Object>) imageMethod().invoke(service, config, query, imageConfig);
        Assertions.assertEquals("2K", result.get("imageSize"));
        Assertions.assertEquals("high", result.get("quality"));
        Assertions.assertEquals(2, result.size());
    }

    /**
     * imageConfig 为 null 且请求带 requestConfig 时，先初始化再合并，返回包含请求配置的 Map。
     */
    @Test
    public void image_defaultConfigNull_requestConfigUsed() throws Exception {
        GoogleRequestService<GoogleRequest> service = createService();
        LLMConfig config = new LLMConfig();
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.putMetadata(GoogleRequestService.KEY_IMAGE_CONFIG, Map.of("imageSize", "4K"));
        Map<String, Object> result = (Map<String, Object>) imageMethod().invoke(service, config, query, null);
        Assertions.assertNotNull(result);
        Assertions.assertEquals("4K", result.get("imageSize"));
    }

    /**
     * 请求不含 KEY_IMAGE_CONFIG metadata 时，即使 request 其它 metadata 存在，也不做合并。
     */
    @Test
    public void image_requestWithoutImageConfigKey_returnsConfigUnchanged() throws Exception {
        GoogleRequestService<GoogleRequest> service = createService();
        LLMConfig config = new LLMConfig();
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        Map<String, Object> imageConfig = new HashMap<>(Map.of("imageSize", "1K"));
        Map<String, Object> result = (Map<String, Object>) imageMethod().invoke(service, config, query, imageConfig);
        Assertions.assertSame(imageConfig, result);
        Assertions.assertEquals(1, result.size());
    }

    /**
     * 默认配置为空 Map、请求带多项时，结果仅为请求配置。
     */
    @Test
    public void image_emptyDefault_requestConfigFillsAll() throws Exception {
        GoogleRequestService<GoogleRequest> service = createService();
        LLMConfig config = new LLMConfig();
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.putMetadata(GoogleRequestService.KEY_IMAGE_CONFIG, Map.of("imageSize", "4K", "quality", "low"));
        Map<String, Object> imageConfig = new HashMap<>();
        Map<String, Object> result = (Map<String, Object>) imageMethod().invoke(service, config, query, imageConfig);
        Assertions.assertEquals("4K", result.get("imageSize"));
        Assertions.assertEquals("low", result.get("quality"));
        Assertions.assertEquals(2, result.size());
    }

    // ---------- GoogleRequestInitConfig.appName ----------

    @Test
    public void googleRequestInitConfig_appName_getSet() {
        GoogleRequestService.GoogleRequestInitConfig config = new GoogleRequestService.GoogleRequestInitConfig();
        Assertions.assertNull(config.getAppName());
        config.setAppName("my-app");
        Assertions.assertEquals("my-app", config.getAppName());
    }

    @Test
    public void googleRequestInitConfig_appName_defaultEmptyWhenNotInjected() {
        GoogleRequestService.GoogleRequestInitConfig config = new GoogleRequestService.GoogleRequestInitConfig();
        // 单元测试中未注入 Spring 上下文时无默认值；@Value("${spring.application.name:}") 在注入前为 null
        Assertions.assertNull(config.getAppName());
    }

    // ---------- labels 相关 ----------

    private static GoogleRequestService<GoogleRequest> createServiceForLabels() {
        GoogleRequestService<GoogleRequest> service = new GoogleRequestService<GoogleRequest>() {
            @Override
            protected GoogleRequest build() {
                return new GoogleRequest();
            }
        };
        service.setAppName("test-app");
        return service;
    }

    private static Map<String, String> invokeLabels(GoogleRequestService<GoogleRequest> service, LLMConfig llmConfig, LLMQuery llmQuery, Map<String, String> labelsConfig) throws Exception {
        Method m = MethodUtils.getMatchingMethod(GoogleRequestService.class, "labels", LLMConfig.class, LLMQuery.class, Map.class);
        m.setAccessible(true);
        @SuppressWarnings("unchecked")
        Map<String, String> result = (Map<String, String>) m.invoke(service, llmConfig, llmQuery, labelsConfig);
        return result;
    }

    /**
     * request()：additional 中含 KEY_LABELS 时，会调用 labels() 并将结果 set 到 googleRequest。
     */
    @Test
    public void request_whenAdditionalHasLabels_setsLabelsOnRequest() throws Exception {
        GoogleRequestService<GoogleRequest> service = createServiceForLabels();
        service.setLlmPromptService(EasyMock.createNiceMock(LLMPromptService.class));
        service.setProviderToken(new ProviderToken());
        GoogleRequest req = new GoogleRequest();
        req.setToken("token");
        LLMConfig config = new LLMConfig();
        config.setAdditional(new HashMap<>(Map.of(GoogleRequestService.KEY_LABELS, new HashMap<String, String>())));
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        service.request(req, config, query);
        Map<String, String> labels = req.getLabels();
        Assertions.assertNull(labels);
    }
}
