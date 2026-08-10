package ai.open.right.workflow.flow.llm.provider.minimax;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallData;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

/**
 * MiniMaxRouter 单元测试。
 */
public class MiniMaxRouterTest {

    private MiniMaxRouter miniMaxRouter;

    @BeforeEach
    public void setUp() {
        miniMaxRouter = new MiniMaxRouter();
        miniMaxRouter.setUrl("https://api.minimaxi.com/test-url");
    }

    @Test
    public void testUrl_messageMetadataOverridesRequestAndRouterUrl() throws Exception {
        AnthropicRequest request = buildRequest(Map.of("__url", "https://metadata.example.com"));
        request.setUrl("https://request.example.com");

        Assertions.assertEquals("https://metadata.example.com", miniMaxRouter.url(request, new LLMConfig(), "any-token"));
    }

    @Test
    public void testUrl_requestUrlOverridesRouterUrl() throws Exception {
        AnthropicRequest request = buildRequest(new HashMap<String, Object>());
        request.setUrl("https://request.example.com");

        Assertions.assertEquals("https://request.example.com", miniMaxRouter.url(request, new LLMConfig(), "any-token"));
    }

    @Test
    public void testUrl_usesRouterUrlWhenRequestUrlIsEmpty() throws Exception {
        AnthropicRequest request = buildRequest(new HashMap<String, Object>());
        request.setUrl("");

        Assertions.assertEquals("https://api.minimaxi.com/test-url", miniMaxRouter.url(request, new LLMConfig(), "any-token"));
    }

    @Test
    public void testUrl_throwsWhenAllUrlSourcesAreEmpty() {
        miniMaxRouter.setUrl("");
        AnthropicRequest request = buildRequest(new HashMap<String, Object>());

        Assertions.assertThrows(IllegalArgumentException.class, () -> miniMaxRouter.url(request, new LLMConfig(), "any-token"));
    }

    @Test
    public void testBody_usesThinkingFromExtraBody() throws Exception {
        AnthropicRequest request = buildRequest(new HashMap<String, Object>());
        Map<String, Object> thinking = Map.of("type", "adaptive", "budget_tokens", 1024);
        request.setThinking(Map.of("type", "disabled"));
        request.setExtra(ProviderRequestService.KEY_THINKING, thinking);

        Object body = miniMaxRouter.body(request);

        Assertions.assertTrue(body instanceof MiniMaxRouter.MiniMaxMessage);
        MiniMaxRouter.MiniMaxMessage message = (MiniMaxRouter.MiniMaxMessage) body;
        Assertions.assertSame(thinking, message.getThinking());
    }

    @Test
    public void testBody_setsThinkingToNullWhenExtraBodyHasNoThinkingMap() throws Exception {
        AnthropicRequest withoutExtraBody = buildRequest(new HashMap<String, Object>());
        AnthropicRequest withInvalidThinking = buildRequest(new HashMap<String, Object>());
        withInvalidThinking.setExtra(ProviderRequestService.KEY_THINKING, "adaptive");

        Assertions.assertNull(((MiniMaxRouter.MiniMaxMessage) miniMaxRouter.body(withoutExtraBody)).getThinking());
        Assertions.assertNull(((MiniMaxRouter.MiniMaxMessage) miniMaxRouter.body(withInvalidThinking)).getThinking());
    }

    @Test
    public void testMiniMaxContent_funCallWithReason_returnsThinkingThenToolUse() throws Exception {
        ProviderFunCallRequest request = ProviderFunCallRequest.builder()
                .created(10L).id("call_1").name("weather").args(Map.of("city", "Shanghai")).reason("need weather data")
                .build();

        MiniMaxRouter.MiniMaxContent content = new MiniMaxRouter.MiniMaxContent(request);

        Assertions.assertEquals(10L, content.getCreated());
        Assertions.assertEquals("assistant", content.getRole());
        Assertions.assertEquals("call_1", content.getRequestToolCallId());
        Object[] blocks = (Object[]) content.getContent();
        Assertions.assertEquals(2, blocks.length);
        Map<?, ?> thinking = (Map<?, ?>) blocks[0];
        Map<?, ?> toolUse = (Map<?, ?>) blocks[1];
        Assertions.assertEquals("thinking", thinking.get("type"));
        Assertions.assertEquals("need weather data", thinking.get("thinking"));
        Assertions.assertEquals("tool_use", toolUse.get("type"));
        Assertions.assertEquals("call_1", toolUse.get("id"));
        Assertions.assertEquals(Map.of("city", "Shanghai"), toolUse.get("input"));
    }

    @Test
    public void testMiniMaxContent_funCallWithoutReason_returnsOnlyToolUse() throws Exception {
        ProviderFunCallRequest request = ProviderFunCallRequest.builder()
                .created(11L).id("call_2").name("weather").args(Map.of()).reason("")
                .build();

        Object[] blocks = (Object[]) new MiniMaxRouter.MiniMaxContent(request).getContent();

        Assertions.assertEquals(1, blocks.length);
        Assertions.assertEquals("tool_use", ((Map<?, ?>) blocks[0]).get("type"));
    }

    @Test
    public void testMiniMaxContent_historyWithAndWithoutReason_coversBothRoles() throws Exception {
        History assistant = new History();
        assistant.setCreated(20L);
        assistant.setRole(History.ROLE_ASSISTANT);
        assistant.setContent("assistant answer");
        assistant.setReason("assistant thinking");

        MiniMaxRouter.MiniMaxContent assistantContent = new MiniMaxRouter.MiniMaxContent(assistant);
        Object[] assistantBlocks = (Object[]) assistantContent.getContent();
        Assertions.assertEquals("assistant", assistantContent.getRole());
        Assertions.assertEquals(2, assistantBlocks.length);
        Assertions.assertEquals("assistant thinking", ((Map<?, ?>) assistantBlocks[0]).get("thinking"));
        Assertions.assertEquals("text", ((Map<?, ?>) assistantBlocks[1]).get("type"));
        Assertions.assertEquals("assistant answer", ((Map<?, ?>) assistantBlocks[1]).get("text"));

        History user = new History();
        user.setCreated(21L);
        user.setRole(History.ROLE_USER);
        user.setContent("user question");

        MiniMaxRouter.MiniMaxContent userContent = new MiniMaxRouter.MiniMaxContent(user);
        Object[] userBlocks = (Object[]) userContent.getContent();
        Assertions.assertEquals("user", userContent.getRole());
        Assertions.assertEquals(1, userBlocks.length);
        Assertions.assertEquals("user question", ((Map<?, ?>) userBlocks[0]).get("text"));
    }

    @Test
    public void testMiniMaxMessage_usesOverridesForFunCallAndChatHistory() throws Exception {
        AnthropicRequest funCallRequest = buildRequest(new HashMap<String, Object>());
        ProviderFunCallData funCallData = new ProviderFunCallData();
        funCallData.addFunCall(
                ProviderFunCallRequest.builder().created(1L).id("call_3").name("tool").args(Map.of()).reason("tool thinking").build(),
                ProviderFunCallResponse.builder().created(2L).id("call_3").name("tool").response("tool result").build());
        funCallRequest.setFunCallData(funCallData);

        MiniMaxRouter.MiniMaxMessage funCallMessage = (MiniMaxRouter.MiniMaxMessage) miniMaxRouter.body(funCallRequest);
        Assertions.assertInstanceOf(MiniMaxRouter.MiniMaxContent.class, funCallMessage.getMessages().getFirst());
        Assertions.assertEquals(2, ((Object[]) funCallMessage.getMessages().getFirst().getContent()).length);

        AnthropicRequest chatRequest = buildRequest(new HashMap<String, Object>());
        History history = new History();
        history.setCreated(1L);
        history.setRole(History.ROLE_ASSISTANT);
        history.setContent("previous answer");
        history.setReason("previous thinking");
        chatRequest.getMessage().addHistory(history);

        MiniMaxRouter.MiniMaxMessage chatMessage = (MiniMaxRouter.MiniMaxMessage) miniMaxRouter.body(chatRequest);
        Assertions.assertInstanceOf(MiniMaxRouter.MiniMaxContent.class, chatMessage.getMessages().getFirst());
        Assertions.assertEquals(2, ((Object[]) chatMessage.getMessages().getFirst().getContent()).length);
    }

    /**
     * 测试内部类 InitConfig 的 Bean 创建逻辑。
     * 验证属性是否通过 BeanUtils 正确拷贝到 MiniMaxRouter 实例中。
     */
    @Test
    public void testInitConfig() throws Exception {
        MiniMaxRouter.InitConfig initConfig = new MiniMaxRouter.InitConfig();
        String expectedUrl = "https://api.minimaxi.com/anthropic/v1/messages";
        initConfig.setUrl(expectedUrl);

        MiniMaxRouter router = initConfig.miniMaxRouter();

        Assertions.assertNotNull(router, "Created router should not be null");
        Assertions.assertEquals(expectedUrl, router.getUrl(), "URL property should be copied from InitConfig");
    }

    private AnthropicRequest buildRequest(Map<String, Object> metadata) {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery(metadata)));
        return request;
    }
}
