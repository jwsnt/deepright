package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMPromptService;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.ProviderToken;
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
public class SeedreamRequestServiceTest {

    @Mock
    private LLMPromptService llmPromptService;

    @InjectMocks
    private SeedreamRequestService seedRequestService;

    @BeforeEach
    public void setUp() {
        // MockitoExtension 会自动处理 @Mock 和 @InjectMocks 的初始化
    }

    /**
     * 测试 build 方法是否正确创建 AnthropicRequest
     */
    @Test
    public void testBuild() throws Exception {
        SeedreamRequest request = seedRequestService.build();
        Assertions.assertNotNull(request, "Build should return a non-null SeedRequest");
        Assertions.assertTrue(request instanceof SeedreamRequest, "Request should be an instance of SeedRequest");
    }

    /**
     * 测试 request 方法的逻辑，包括 extraBody 设置和 model/token 的默认值填充
     */
    @Test
    public void testRequest() throws Exception {
        // 准备 Service 默认配置
        seedRequestService.setModel("claude-3-opus-20240229");
        seedRequestService.setToken("sk-ant-service-token");

        // 准备 LLMConfig 和 LLMQuery
        LLMConfig llmConfig = new LLMConfig();
        Map<String, Object> additional = new HashMap<>();
        Map<String, Object> extraBody = new HashMap<>();
        Map<String, Object> thinking = new HashMap<>();
        extraBody.put("max_tokens", 4096);
        additional.put(ProviderRequestService.KEY_EXTRA_BODY, extraBody);
        additional.put(ProviderRequestService.KEY_THINKING, thinking);
        llmConfig.setAdditional(additional);

        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();

        // 场景 1: SeedRequest 中 model 和 token 为空，应使用 Service 的默认值
        SeedreamRequest request1 = new SeedreamRequest();
        seedRequestService.setProviderToken(new ProviderToken());
        seedRequestService.request(request1, llmConfig, llmQuery);
        Assertions.assertEquals("claude-3-opus-20240229", request1.getModel(), "Should use service model");
        Assertions.assertEquals("Bearer sk-ant-service-token", request1.getToken(), "Should use service token with Bearer prefix");

        // 场景 2: AnthropicRequest 中已有 model 和 token，不应被 Service 默认值覆盖
        SeedreamRequest request2 = new SeedreamRequest();
        request2.setModel("claude-3-sonnet-20240229");
        request2.setToken("sk-ant-request-token");
        seedRequestService.request(request2, llmConfig, llmQuery);

        Assertions.assertEquals("claude-3-opus-20240229", request2.getModel(), "Should keep request model");
        Assertions.assertEquals("Bearer sk-ant-service-token", request2.getToken(), "Should use service token with Bearer prefix");
    }

    @Test
    public void testRequestDisablesUnsupportedSequentialGenerationForProModel() throws Exception {
        seedRequestService.setModel("doubao-seedream-5-0-pro-260628");
        seedRequestService.setToken("seedream-token");
        seedRequestService.setProviderToken(new ProviderToken());

        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setStream(true);
        Map<String, Object> additional = new HashMap<String, Object>();
        additional.put(SeedreamRequestService.KEY_SEQUENTIAL_IMAGE_GENERATION, "enabled");
        llmConfig.setAdditional(additional);

        SeedreamRequest request = new SeedreamRequest();
        seedRequestService.request(request, llmConfig, ObjectBuilder.buildLLMQuery());

        Assertions.assertNull(request.getSequential(), "Pro models must omit sequential_image_generation");
        Assertions.assertFalse(request.getStream(), "Pro models do not support streaming image generation");
        String body = JsonUtils.write(new SeedreamRouter.SeedreamMessage(request));
        Assertions.assertFalse(body.contains("\"sequential_image_generation\":"), "Downstream request body must omit sequential_image_generation");
        Assertions.assertTrue(body.contains("\"stream\":false"), "Downstream request body must disable streaming");
    }

    /**
     * 测试内部类 InitConfig 的初始化逻辑
     */
    @Test
    public void testInitConfig() throws Exception {
        SeedreamRequestService.InitConfig initConfig = new SeedreamRequestService.InitConfig();
        initConfig.setModel("claude-v1-config");
        initConfig.setToken("token-from-config");
        initConfig.setFunCallTimeout(10000);

        // 执行 Bean 创建方法
        SeedreamRequestService service = initConfig.seedreamRequestService();

        Assertions.assertNotNull(service, "Service should be created by InitConfig");
        Assertions.assertEquals("claude-v1-config", service.getModel(), "Model should be copied from config");
        Assertions.assertEquals("token-from-config", service.getToken(), "Token should be copied from config");
        Assertions.assertEquals(Integer.valueOf(10000), service.getFunCallTimeout(), "Timeout should be copied from config");
    }

    @Test
    public void testGetModel() throws Exception {
        seedRequestService.setModel("seedream-4-5");
        Assertions.assertEquals("seedream-4-5", seedRequestService.getModel(ObjectBuilder.buildWorkflowTask()));
    }
}
