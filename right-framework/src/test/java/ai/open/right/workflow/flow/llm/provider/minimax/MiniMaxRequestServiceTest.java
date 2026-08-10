package ai.open.right.workflow.flow.llm.provider.minimax;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

/**
 * MiniMaxRequestService 单元测试
 */
@ExtendWith(MockitoExtension.class)
class MiniMaxRequestServiceTest {

    private TestMiniMaxRequestService miniMaxRequestService;

    @Mock
    private LLMPromptService llmPromptService;

    @BeforeEach
    void setUp() {
        miniMaxRequestService = new TestMiniMaxRequestService();
        // 注入 Mock 的 LLMPromptService 以防止 ProviderRequestService.addPrompt 抛出 NPE
        ReflectionTestUtils.setField(miniMaxRequestService, "llmPromptService", llmPromptService);
    }

    /**
     * 测试 build 方法是否正确创建 AnthropicRequest (MiniMax 继承自 Anthropic)
     */
    @Test
    void testBuild() throws Exception {
        AnthropicRequest request = miniMaxRequestService.build();
        Assertions.assertNotNull(request, "Build should return a non-null AnthropicRequest");
        Assertions.assertTrue(request instanceof AnthropicRequest, "Request should be an instance of AnthropicRequest");
    }

    /**
     * 测试 request 方法的逻辑，包括 extraBody 设置和 model/token 的默认值填充
     */
    @Test
    void testRequest() throws Exception {
        // 准备 Service 默认配置
        miniMaxRequestService.setModel("MiniMax-M2.1");
        miniMaxRequestService.setToken("minimax-service-token");

        // 准备 LLMConfig 和 LLMQuery
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> additional = new HashMap<>();
        Map<String, Object> extraBody = new HashMap<>();
        extraBody.put("temperature", 0.7);
        additional.put(ProviderRequestService.KEY_EXTRA_BODY, extraBody);
        llmConfig.setAdditional(additional);

        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();

        // 场景 1: AnthropicRequest 中 model 和 token 为空，应使用 Service 的默认值
        AnthropicRequest request1 = new AnthropicRequest();
        miniMaxRequestService.setProviderToken(new ProviderToken());
        miniMaxRequestService.request(request1, llmConfig, llmQuery);
        Assertions.assertEquals("MiniMax-M2.1", request1.getModel(), "Should use service model");
        Assertions.assertEquals("minimax-service-token", request1.getToken(), "Should use service token");
        Assertions.assertNull(request1.getExtraBody(), "MiniMax ignores generic extraBody configuration");

        // 场景 2: AnthropicRequest 中已有 model 和 token，不应被 Service 默认值覆盖
        AnthropicRequest request2 = new AnthropicRequest();
        request2.setModel("MiniMax-Custom-Model");
        request2.setToken("minimax-request-token");
        miniMaxRequestService.request(request2, llmConfig, llmQuery);

        Assertions.assertEquals("MiniMax-M2.1", request2.getModel(), "Should keep request model");
        Assertions.assertEquals("minimax-service-token", request2.getToken(), "Should keep request token");
    }

    /**
     * 测试内部类 InitConfig 的初始化逻辑
     */
    @Test
    void testInitConfig() throws Exception {
        MiniMaxRequestService.InitConfig initConfig = new MiniMaxRequestService.InitConfig();
        initConfig.setModel("minimax-config-model");
        initConfig.setToken("token-from-config");
        initConfig.setFunCallTimeout(5000);

        // 执行 Bean 创建方法
        MiniMaxRequestService service = initConfig.miniMaxRequestService();
        // 注入 Mock 对象
        ReflectionTestUtils.setField(service, "llmPromptService", llmPromptService);

        Assertions.assertNotNull(service, "Service should be created by InitConfig");
        Assertions.assertEquals("minimax-config-model", service.getModel(), "Model should be copied from config");
        Assertions.assertEquals("token-from-config", service.getToken(), "Token should be copied from config");
        Assertions.assertEquals(Integer.valueOf(5000), service.getFunCallTimeout(), "Timeout should be copied from config");
    }

    @Test
    void testGetModel() throws Exception {
        miniMaxRequestService.setModel("MiniMax-M2.1");
        Assertions.assertEquals("MiniMax-M2.1", miniMaxRequestService.getModel(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    void reasoningShouldPreferDisabledThinkingFromQueryMetadata() throws Exception {
        Map<String, Object> queryThinking = new HashMap<>();
        queryThinking.put("type", "disabled");
        queryThinking.put("custom", "value");
        Map<String, Object> queryMetadata = new HashMap<>();
        queryMetadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, queryThinking);

        LLMConfig llmConfig = new LLMConfig();
        llmConfig.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"));
        AnthropicRequest request = new AnthropicRequest();

        miniMaxRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQuery(queryMetadata));

        Assertions.assertSame(queryThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
    }

    @Test
    void reasoningShouldConvertNonDisabledQueryThinkingToAdaptive() throws Exception {
        Map<String, Object> queryMetadata = new HashMap<>();
        queryMetadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"));

        LLMConfig llmConfig = new LLMConfig();
        llmConfig.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "disabled"));
        AnthropicRequest request = new AnthropicRequest();

        miniMaxRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQuery(queryMetadata));

        Assertions.assertEquals(MiniMaxRequestService.THINK_CONFIG, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
    }

    @Test
    void reasoningShouldFallbackToDisabledThinkingFromConfigWhenQueryThinkingIsEmpty() throws Exception {
        Map<String, Object> queryMetadata = new HashMap<>();
        queryMetadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, Collections.emptyMap());
        Map<String, Object> configThinking = new HashMap<>();
        configThinking.put("type", "DISABLED");

        LLMConfig llmConfig = new LLMConfig();
        llmConfig.getAdditional().put(ProviderRequestService.KEY_THINKING, configThinking);
        AnthropicRequest request = new AnthropicRequest();

        miniMaxRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQuery(queryMetadata));

        Assertions.assertSame(configThinking, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
    }

    @Test
    void reasoningShouldConvertNonDisabledConfigThinkingToAdaptive() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.getAdditional().put(ProviderRequestService.KEY_THINKING, Collections.singletonMap("type", "enabled"));
        AnthropicRequest request = new AnthropicRequest();

        miniMaxRequestService.reasoning(request, llmConfig, ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assertions.assertEquals(MiniMaxRequestService.THINK_CONFIG, request.getExtraBody().get(ProviderRequestService.KEY_THINKING));
    }

    @Test
    void reasoningShouldNotSetThinkingWhenNeitherQueryNorConfigProvidesIt() throws Exception {
        AnthropicRequest request = new AnthropicRequest();

        miniMaxRequestService.reasoning(request, new LLMConfig(), ObjectBuilder.buildLLMQueryWithEmptyMetadata());

        Assertions.assertNull(request.getExtraBody());
    }

    /**
     * 内部测试类，通过继承暴露受保护的方法
     */
    private static class TestMiniMaxRequestService extends MiniMaxRequestService {
        @Override
        public AnthropicRequest build() throws Exception {
            return super.build();
        }

        @Override
        public void request(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            super.request(request, llmConfig, llmQuery);
        }

        @Override
        public void reasoning(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            super.reasoning(request, llmConfig, llmQuery);
        }
    }
}
