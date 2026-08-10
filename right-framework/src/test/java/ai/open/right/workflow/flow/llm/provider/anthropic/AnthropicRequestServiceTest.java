package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
import org.apache.commons.collections.MapUtils;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.HashMap;
import java.util.Map;

/**
 * AnthropicRequestService 单元测试
 */
@ExtendWith(MockitoExtension.class)
public class AnthropicRequestServiceTest {

    @Mock
    private LLMPromptService llmPromptService;

    @InjectMocks
    private AnthropicRequestService anthropicRequestService;

    @BeforeEach
    public void setUp() {
        // MockitoExtension 会自动处理 @Mock 和 @InjectMocks 的初始化
    }

    /**
     * 测试 build 方法是否正确创建 AnthropicRequest
     */
    @Test
    public void testBuild() throws Exception {
        AnthropicRequest request = anthropicRequestService.build();
        Assertions.assertNotNull(request, "Build should return a non-null AnthropicRequest");
        Assertions.assertTrue(request instanceof AnthropicRequest, "Request should be an instance of AnthropicRequest");
    }

    /**
     * 测试 request 方法的逻辑，包括 thinking、默认缓存及 model/token 的默认值填充
     */
    @Test
    public void testRequest() throws Exception {
        // 准备 Service 默认配置
        anthropicRequestService.setModel("claude-3-opus-20240229");
        anthropicRequestService.setToken("sk-ant-service-token");

        // 准备 LLMConfig 和 LLMQuery
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> additional = new HashMap<>();
        Map<String, Object> thinking = Map.of("type", "enabled", "budget_tokens", 1024);
        additional.put(ProviderRequestService.KEY_THINKING, thinking);
        llmConfig.setAdditional(additional);

        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();

        // 场景 1: AnthropicRequest 中 model 和 token 为空，应使用 Service 的默认值
        AnthropicRequest request1 = new AnthropicRequest();
        anthropicRequestService.setProviderToken(new ProviderToken());
        anthropicRequestService.request(request1, llmConfig, llmQuery);
        Assertions.assertEquals(AnthropicRequestService.THINK_CONFIG, request1.getThinking());
        Assertions.assertEquals("claude-3-opus-20240229", request1.getModel(), "Should use service model");
        Assertions.assertEquals("sk-ant-service-token", request1.getToken(), "Should use service token");
        Assertions.assertEquals(AnthropicRequestService.CACHE_CONTROL, request1.getCacheControl(), "Should enable default cache control");
        Assertions.assertNull(request1.getExtraBody(), "Should not use extraBody for cache control");

        // 场景 2: AnthropicRequest 中已有 model 和 token，不应被 Service 默认值覆盖
        AnthropicRequest request2 = new AnthropicRequest();
        request2.setModel("claude-3-sonnet-20240229");
        request2.setToken("sk-ant-request-token");
        anthropicRequestService.request(request2, llmConfig, llmQuery);

        Assertions.assertEquals("claude-3-opus-20240229", request2.getModel(), "Should keep request model");
        Assertions.assertEquals("sk-ant-service-token", request2.getToken(), "Should keep request token");
    }

    /**
     * 测试内部类 InitConfig 的初始化逻辑
     */
    @Test
    public void testInitConfig() throws Exception {
        AnthropicRequestService.InitConfig initConfig = new AnthropicRequestService.InitConfig();
        initConfig.setModel("claude-v1-config");
        initConfig.setToken("token-from-config");
        initConfig.setFunCallTimeout(10000);

        // 执行 Bean 创建方法
        AnthropicRequestService service = initConfig.anthropicRequestService();

        Assertions.assertNotNull(service, "Service should be created by InitConfig");
        Assertions.assertEquals("claude-v1-config", service.getModel(), "Model should be copied from config");
        Assertions.assertEquals("token-from-config", service.getToken(), "Token should be copied from config");
        Assertions.assertEquals(Integer.valueOf(10000), service.getFunCallTimeout(), "Timeout should be copied from config");
    }

    @Test
    public void testGetModel() throws Exception {
        anthropicRequestService.setModel("claude-3-opus");
        Assertions.assertEquals("claude-3-opus", anthropicRequestService.getModel(ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testExtra_enablesDefaultCacheControl() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new ai.open.right.workflow.flow.llm.MessageDelegate(ObjectBuilder.buildLLMQuery()));
        LLMConfig llmConfig = new LLMConfig();

        anthropicRequestService.extra(request, llmConfig, ObjectBuilder.buildLLMQuery());

        Assertions.assertEquals(AnthropicRequestService.CACHE_CONTROL, request.getCacheControl());
        Assertions.assertNull(request.getExtraBody());
    }

    @Test
    public void testExtra_ignoresCustomCacheControl() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new ai.open.right.workflow.flow.llm.MessageDelegate(ObjectBuilder.buildLLMQuery()));
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> additional = new HashMap<>();
        Map<String, Object> extraBody = new HashMap<>();
        Map<String, Object> cacheControl = new HashMap<>();
        cacheControl.put("type", "persistent");
        extraBody.put("cache_control", cacheControl);
        extraBody.put("k", "v");
        additional.put(ProviderRequestService.KEY_EXTRA_BODY, extraBody);
        llmConfig.setAdditional(additional);

        anthropicRequestService.extra(request, llmConfig, ObjectBuilder.buildLLMQuery());

        Assertions.assertEquals(AnthropicRequestService.CACHE_CONTROL, request.getCacheControl());
        Assertions.assertNull(request.getExtraBody());
    }
}
