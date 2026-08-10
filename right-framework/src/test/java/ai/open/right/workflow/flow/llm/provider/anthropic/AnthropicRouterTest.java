package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.*;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallData;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import com.google.common.collect.ImmutableMap;
import org.apache.http.client.methods.HttpPost;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.slf4j.LoggerFactory;

import java.lang.reflect.Method;
import java.util.*;
import java.util.stream.Collectors;

/**
 * AnthropicRouter 单测，覆盖 reConfig, url, body 以及内部类。
 */
public class AnthropicRouterTest {

    private AnthropicRouter anthropicRouter;

    @BeforeEach
    public void setUp() {
        anthropicRouter = new AnthropicRouter();
        anthropicRouter.setUrl("http://test-url");
    }

    private AnthropicRequest mockAnthropicRequest() {
        AnthropicRequest request = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(request.getExtraBody()).andReturn(null).anyTimes();
        EasyMock.expect(request.getCacheControl()).andReturn(null).anyTimes();
        return request;
    }

    @Test
    public void testReConfig() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        request.setTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        LLMConfig config = EasyMock.createMock(LLMConfig.class);
        Message message = EasyMock.createMock(Message.class);
        HttpPost httpPost = new HttpPost("http://test-url");
        request.setFunCallTimeout(EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getTimeout()).andReturn(2000).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("test-token").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        // 修复 1: mock getMessage()
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        // 修复 1: 为 message mock 添加 isFromFunCall() 的 mock
        EasyMock.expect(message.isFromFunCall()).andReturn(false).anyTimes();
        // 修复: 添加 getUpstream() 和 getTimeout() 的 mock
        EasyMock.expect(message.getUpstream()).andReturn(null).anyTimes();
        EasyMock.expect(config.getTimeout(EasyMock.anyInt())).andReturn(60000).anyTimes();

        EasyMock.replay(request, config, message);
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        anthropicRouter.setTimeout(1000);
        anthropicRouter.setTimeoutRate(2.0D);
        anthropicRouter.setHttpClientConfig(httpClientConfig);
        anthropicRouter.reConfig(request, config, httpPost);

        // 验证 Header 是否正确添加
        Assertions.assertEquals("test-token", httpPost.getFirstHeader("X-Api-Key").getValue());
        EasyMock.verify(request, config, message);
    }

    @Test
    public void testUrl() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getUrl()).andReturn("https://www.google.com").anyTimes();
        LLMConfig config = EasyMock.createMock(LLMConfig.class);
        EasyMock.replay(request, config);
        String result = anthropicRouter.url(request, config, "test");
        Assertions.assertEquals("https://www.google.com", result);
        EasyMock.verify(request, config);
    }

    @Test
    public void testBody() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(1L);
        nettyRequest.setQuery("hello");

        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(Message.build(llmQuery));
        request.setMaxTokens(100);
        request.setTemperature(0.7);
        request.setStream(true);
        request.setModel("claude-3");
        request.setTopP(0.9);
        request.setPrompt("system prompt");

        Object body = anthropicRouter.body(request);
        Assertions.assertTrue(body instanceof AnthropicRouter.AnthropicMessage);
        AnthropicRouter.AnthropicMessage anthropicMessage = (AnthropicRouter.AnthropicMessage) body;

        // 验证字段赋值
        Assertions.assertEquals(Integer.valueOf(100), anthropicMessage.getMaxTokens());
        Assertions.assertEquals("claude-3", anthropicMessage.getModel());
        Assertions.assertEquals("system prompt", anthropicMessage.getSystem());
        Assertions.assertEquals(1, anthropicMessage.getMessages().size());
        Assertions.assertEquals("hello", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("user", anthropicMessage.getMessages().get(0).getRole());
    }

    @Test
    public void testBody_withCacheControl_setsSystemCacheControlBlock() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setMaxTokens(100);
        request.setModel("claude-3");
        request.setPrompt("system prompt");
        request.setStream(true);
        request.setCacheControl(ImmutableMap.of("type", "ephemeral"));

        Object body = anthropicRouter.body(request);

        Assertions.assertTrue(body instanceof AnthropicRouter.AnthropicMessage);
        AnthropicRouter.AnthropicMessage anthropicMessage = (AnthropicRouter.AnthropicMessage) body;
        Assertions.assertTrue(anthropicMessage.getSystem() instanceof List);
        List<?> systemBlocks = (List<?>) anthropicMessage.getSystem();
        Assertions.assertEquals(1, systemBlocks.size());
        Assertions.assertTrue(systemBlocks.get(0) instanceof Map);
        Map<?, ?> systemBlock = (Map<?, ?>) systemBlocks.get(0);
        Assertions.assertEquals("text", systemBlock.get("type"));
        Assertions.assertEquals("system prompt", systemBlock.get("text"));
        Assertions.assertEquals("ephemeral", ((Map<?, ?>) systemBlock.get("cache_control")).get("type"));
    }

    @Test
    public void testAnthropicMessageWithHistory() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        History history = EasyMock.createMock(History.class);
        EasyMock.expect(history.getCreated()).andReturn(1L).anyTimes();
        EasyMock.expect(history.isFunction(0)).andReturn(true).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();

        // 模拟历史记录
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(history)).anyTimes();
        EasyMock.expect(history.isRole(History.ROLE_USER)).andReturn(true).anyTimes();
        EasyMock.expect(history.getContent()).andReturn("history content").anyTimes();
        EasyMock.expect(history.getRole()).andReturn(History.ROLE_USER).anyTimes();

        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("hello").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();

        EasyMock.replay(request, message, history);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        // 验证消息列表包含历史记录和当前 Query
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        Assertions.assertEquals("history content", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("hello", anthropicMessage.getMessages().get(1).getContent());
        // 验证默认 MaxToken
        Assertions.assertEquals(Integer.valueOf(AnthropicRouter.MAX_TOKEN), anthropicMessage.getMaxTokens());

        EasyMock.verify(request, message, history);
    }

    @Test
    public void testAnthropicMessageSystemIsTopLevelWithHistory() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(100L);
        nettyRequest.setQuery("CURRENT_QUERY");

        History history = new History();
        history.setFunction(History.FUN_CHAT);
        history.setCreated(50L);
        history.setContent("HISTORY_CONTENT");
        history.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistory(history);

        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(message);
        request.setPrompt("SYSTEM_PROMPT");

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals("SYSTEM_PROMPT", anthropicMessage.getSystem());
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        Assertions.assertEquals("HISTORY_CONTENT", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("CURRENT_QUERY", anthropicMessage.getMessages().get(1).getContent());
    }

    @Test
    public void testAnthropicFunCallResponseFollowsMatchingRequest() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(400L);
        nettyRequest.setQuery("CURRENT_QUERY");

        ProviderFunCallRequest request = new ProviderFunCallRequest() {
            @Override
            public Long getCreated() {
                return 200L;
            }
        };
        request.setId("request_1");
        request.setName("tool_x");
        request.setArgs(ImmutableMap.of("k", "v"));

        ProviderFunCallResponse response = new ProviderFunCallResponse() {
            @Override
            public Long getCreated() {
                return 100L;
            }
        };
        response.setId("response_1");
        response.setName("tool_x");
        response.setResponse("tool_result");

        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        providerFunCallData.addFunCall(request, response);

        AnthropicRequest anthropicRequest = new AnthropicRequest();
        anthropicRequest.setMessage(Message.build(llmQuery));
        anthropicRequest.setFunCallData(providerFunCallData);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(anthropicRequest);
        Assertions.assertEquals(3, anthropicMessage.getMessages().size());
        Assertions.assertEquals("assistant", anthropicMessage.getMessages().get(0).getRole());
        Object[] toolUseContent = (Object[]) anthropicMessage.getMessages().get(0).getContent();
        Assertions.assertEquals("tool_use", ((Map<?, ?>) toolUseContent[0]).get("type"));
        Assertions.assertEquals("request_1", ((Map<?, ?>) toolUseContent[0]).get("id"));
        Assertions.assertEquals("user", anthropicMessage.getMessages().get(1).getRole());
        Object[] toolResultContent = (Object[]) anthropicMessage.getMessages().get(1).getContent();
        Assertions.assertEquals("tool_result", ((Map<?, ?>) toolResultContent[0]).get("type"));
        Assertions.assertEquals("request_1", ((Map<?, ?>) toolResultContent[0]).get("tool_use_id"));
        Assertions.assertEquals("user", anthropicMessage.getMessages().get(2).getRole());
        Assertions.assertEquals("CURRENT_QUERY", anthropicMessage.getMessages().get(2).getContent());
    }

    @Test
    public void testAnthropicFunCallResponseFollowsMatchingRequestWithSameCreated() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(300L);
        nettyRequest.setQuery("CURRENT_QUERY");

        ProviderFunCallRequest request = new ProviderFunCallRequest() {
            @Override
            public Long getCreated() {
                return 100L;
            }
        };
        request.setId("same_id");
        request.setName("tool_x");
        request.setArgs(ImmutableMap.of("k", "v"));

        ProviderFunCallResponse response = new ProviderFunCallResponse() {
            @Override
            public Long getCreated() {
                return 100L;
            }
        };
        response.setId("same_id");
        response.setName("tool_x");
        response.setResponse("tool_result");

        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        providerFunCallData.addFunCall(request, response);

        AnthropicRequest anthropicRequest = new AnthropicRequest();
        anthropicRequest.setMessage(Message.build(llmQuery));
        anthropicRequest.setFunCallData(providerFunCallData);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(anthropicRequest);
        Assertions.assertEquals(3, anthropicMessage.getMessages().size());
        Object[] toolUseContent = (Object[]) anthropicMessage.getMessages().get(0).getContent();
        Object[] toolResultContent = (Object[]) anthropicMessage.getMessages().get(1).getContent();
        Assertions.assertEquals("same_id", ((Map<?, ?>) toolUseContent[0]).get("id"));
        Assertions.assertEquals("same_id", ((Map<?, ?>) toolResultContent[0]).get("tool_use_id"));
    }

    @Test
    public void testAnthropicFunCallResponsesStayWithOwnIds() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(500L);
        nettyRequest.setQuery("CURRENT_QUERY");

        ProviderFunCallRequest requestA = new ProviderFunCallRequest() {
            @Override
            public Long getCreated() {
                return 100L;
            }
        };
        requestA.setId("id_a");
        requestA.setName("tool_a");
        requestA.setArgs(ImmutableMap.of("k", "a"));

        ProviderFunCallResponse responseA = new ProviderFunCallResponse() {
            @Override
            public Long getCreated() {
                return 350L;
            }
        };
        responseA.setId("other_a");
        responseA.setName("tool_a");
        responseA.setResponse("result_a");

        ProviderFunCallRequest requestB = new ProviderFunCallRequest() {
            @Override
            public Long getCreated() {
                return 200L;
            }
        };
        requestB.setId("id_b");
        requestB.setName("tool_b");
        requestB.setArgs(ImmutableMap.of("k", "b"));

        ProviderFunCallResponse responseB = new ProviderFunCallResponse() {
            @Override
            public Long getCreated() {
                return 50L;
            }
        };
        responseB.setId("other_b");
        responseB.setName("tool_b");
        responseB.setResponse("result_b");

        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        providerFunCallData.addFunCall(requestA, responseA);
        providerFunCallData.addFunCall(requestB, responseB);

        AnthropicRequest anthropicRequest = new AnthropicRequest();
        anthropicRequest.setMessage(Message.build(llmQuery));
        anthropicRequest.setFunCallData(providerFunCallData);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(anthropicRequest);
        Assertions.assertEquals(5, anthropicMessage.getMessages().size());
        Object[] toolUseA = (Object[]) anthropicMessage.getMessages().get(0).getContent();
        Object[] toolResultA = (Object[]) anthropicMessage.getMessages().get(1).getContent();
        Object[] toolUseB = (Object[]) anthropicMessage.getMessages().get(2).getContent();
        Object[] toolResultB = (Object[]) anthropicMessage.getMessages().get(3).getContent();
        Assertions.assertEquals("id_a", ((Map<?, ?>) toolUseA[0]).get("id"));
        Assertions.assertEquals("id_a", ((Map<?, ?>) toolResultA[0]).get("tool_use_id"));
        Assertions.assertEquals("id_b", ((Map<?, ?>) toolUseB[0]).get("id"));
        Assertions.assertEquals("id_b", ((Map<?, ?>) toolResultB[0]).get("tool_use_id"));
    }

    // ---------- buildHistories 全分支覆盖 ----------

    @Test
    public void buildHistories_noHistory_onlyQueryMessage() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(1L).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(100).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);
        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(1, anthropicMessage.getMessages().size());
        Assertions.assertEquals("q", anthropicMessage.getMessages().get(0).getContent());
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_emptyList_onlyQueryMessage() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList()).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);
        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(1, anthropicMessage.getMessages().size());
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_FunCallRequestOnly_toolUseAndQuery() throws Exception {
        ProviderFunCallRequest funCallRequest = ProviderFunCallRequest.builder()
                .id("id1").name("tool_a").args(ImmutableMap.of("k", "v")).build();
        String requestJson = JsonUtils.write(funCallRequest);
        History historyQuery = new History();
        historyQuery.setFunction(History.FUN_FUNCALL);
        historyQuery.setType(History.TYPE_QUERY);
        historyQuery.setContent(requestJson);

        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(historyQuery)).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        AnthropicRouter.AnthropicContent first = anthropicMessage.getMessages().get(0);
        Assertions.assertEquals("assistant", first.getRole());
        Object[] content = (Object[]) first.getContent();
        Map<String, Object> toolUse = (Map<String, Object>) content[0];
        Assertions.assertEquals("tool_use", toolUse.get("type"));
        Assertions.assertEquals("tool_a", toolUse.get("name"));
        Assertions.assertEquals("id1", toolUse.get("id"));
        Assertions.assertEquals("q", anthropicMessage.getMessages().get(1).getContent());
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_FunCallResponseOnly_toolResultAndQuery() throws Exception {
        ProviderFunCallResponse funCallResponse = ProviderFunCallResponse.builder()
                .id("id1").response("result").name("tool_a").build();
        String responseJson = JsonUtils.write(funCallResponse);
        History historyAnswer = new History();
        historyAnswer.setFunction(History.FUN_FUNCALL);
        historyAnswer.setType(History.TYPE_ANSWER);
        historyAnswer.setContent(responseJson);

        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(historyAnswer)).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        AnthropicRouter.AnthropicContent first = anthropicMessage.getMessages().get(0);
        Assertions.assertEquals("user", first.getRole());
        Object[] content = (Object[]) first.getContent();
        Map<String, Object> toolResult = (Map<String, Object>) content[0];
        Assertions.assertEquals("tool_result", toolResult.get("type"));
        Assertions.assertEquals("result", toolResult.get("content"));
        Assertions.assertEquals("id1", toolResult.get("tool_use_id"));
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_FunCallRequestAndResponse_toolUseToolResultAndQuery() throws Exception {
        ProviderFunCallRequest funCallRequest = ProviderFunCallRequest.builder()
                .id("id1").name("tool_a").args(ImmutableMap.of("a", "b")).build();
        ProviderFunCallResponse funCallResponse = ProviderFunCallResponse.builder()
                .id("id1").response("ok").name("tool_a").build();
        History hQuery = new History();
        hQuery.setFunction(History.FUN_FUNCALL);
        hQuery.setType(History.TYPE_QUERY);
        hQuery.setContent(JsonUtils.write(funCallRequest));
        History hAnswer = new History();
        hAnswer.setFunction(History.FUN_FUNCALL);
        hAnswer.setType(History.TYPE_ANSWER);
        hAnswer.setContent(JsonUtils.write(funCallResponse));

        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(hQuery, hAnswer)).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(3, anthropicMessage.getMessages().size());
        Assertions.assertEquals("assistant", anthropicMessage.getMessages().get(0).getRole());
        Assertions.assertEquals("user", anthropicMessage.getMessages().get(1).getRole());
        Assertions.assertEquals("q", anthropicMessage.getMessages().get(2).getContent());
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_mixedChatAndFunCall_chatThenToolUseToolResultAndQuery() throws Exception {
        History chat = new History();
        chat.setFunction(History.FUN_CHAT);
        chat.setType(History.TYPE_QUERY);
        chat.setRole(History.ROLE_ASSISTANT);
        chat.setContent("assistant said");
        ProviderFunCallRequest funCallRequest = ProviderFunCallRequest.builder()
                .id("id1").name("tool_x").args(new HashMap<>()).build();
        ProviderFunCallResponse funCallResponse = ProviderFunCallResponse.builder()
                .id("id1").response("done").name("tool_x").build();
        History hQuery = new History();
        hQuery.setFunction(History.FUN_FUNCALL);
        hQuery.setType(History.TYPE_QUERY);
        hQuery.setContent(JsonUtils.write(funCallRequest));
        History hAnswer = new History();
        hAnswer.setFunction(History.FUN_FUNCALL);
        hAnswer.setType(History.TYPE_ANSWER);
        hAnswer.setContent(JsonUtils.write(funCallResponse));

        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(chat, hQuery, hAnswer)).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(4, anthropicMessage.getMessages().size());
        Assertions.assertEquals("assistant", anthropicMessage.getMessages().get(0).getRole());
        Assertions.assertEquals("assistant said", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("assistant", anthropicMessage.getMessages().get(1).getRole());
        Assertions.assertEquals("user", anthropicMessage.getMessages().get(2).getRole());
        Assertions.assertEquals("q", anthropicMessage.getMessages().get(3).getContent());
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_FunCallOnlyTypeQuery_onlyToolUse() throws Exception {
        ProviderFunCallRequest funCallRequest = ProviderFunCallRequest.builder()
                .id("id1").name("t").args(new HashMap<>()).build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_QUERY);
        h.setContent(JsonUtils.write(funCallRequest));

        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(h)).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        Assertions.assertEquals("tool_use", ((Map) ((Object[]) anthropicMessage.getMessages().get(0).getContent())[0]).get("type"));
        EasyMock.verify(request, message);
    }

    @Test
    public void buildHistories_FunCallOnlyTypeAnswer_onlyToolResult() throws Exception {
        ProviderFunCallResponse funCallResponse = ProviderFunCallResponse.builder()
                .id("id1").response("r").name("t").build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_ANSWER);
        h.setContent(JsonUtils.write(funCallResponse));

        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(Long.MAX_VALUE).anyTimes();
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(true).anyTimes();
        EasyMock.expect(message.getHistories()).andReturn(Arrays.asList(h)).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("q").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();
        EasyMock.replay(request, message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        Assertions.assertEquals("tool_result", ((Map) ((Object[]) anthropicMessage.getMessages().get(0).getContent())[0]).get("type"));
        EasyMock.verify(request, message);
    }

    @Test
    public void testAnthropicMessageWithMime() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(1L).anyTimes();
        MediaContext mediaContext = EasyMock.createMock(MediaContext.class);
        AnthropicMedia anthropicMedia = EasyMock.createMock(AnthropicMedia.class);
        EasyMock.expect(mediaContext.getType("inline:image/png")).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();

        // 模拟多媒体上下文 (图片)
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(Arrays.asList(mediaContext)).anyTimes();
        EasyMock.expect(request.getAnthropicMedia()).andReturn(anthropicMedia).anyTimes();
        // 修复 2: 将媒体类型改为以 inline: 开头
        EasyMock.expect(request.getMimeType()).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("what is this?").anyTimes();

        // 修复 3: 确保 mediaContext.getType 返回 "inline:image/png"
        EasyMock.expect(mediaContext.getType()).andReturn("inline:image/png").anyTimes();
        // 修复 2: 将 anthropicMedia.getPrefix 和 getKeyUrl 的期望参数改为 "inline:image/png"
        EasyMock.expect(anthropicMedia.getType("inline:image/png")).andReturn("data:image/png;base64,").anyTimes();
        EasyMock.expect(mediaContext.getData()).andReturn("base64data").anyTimes();

        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();

        EasyMock.replay(request, message, mediaContext, anthropicMedia);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(1, anthropicMessage.getMessages().size());
        List<Map<String, Object>> content = (List<Map<String, Object>>) anthropicMessage.getMessages().get(0).getContent();
        Assertions.assertEquals(2, content.size()); // Text + Image
        Assertions.assertEquals("text", content.get(0).get("type"));
        Assertions.assertEquals("data:image/png;base64,", content.get(1).get("type"));
        Map<String, Object> image = (Map<String, Object>) content.get(1).get("source");
        Assertions.assertEquals("base64data", image.get("data"));
        EasyMock.verify(request, message, mediaContext, anthropicMedia);
    }

    @Test
    public void testAnthropicMessageWithMimeNotInline() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        MediaContext mediaContext = EasyMock.createMock(MediaContext.class);
        EasyMock.expect(mediaContext.getType("inline:image/png")).andReturn("inline:image/png").anyTimes();
        EasyMock.expect(mediaContext.getType("text/plain")).andReturn("text/plain").anyTimes();
        AnthropicMedia anthropicMedia = EasyMock.createMock(AnthropicMedia.class);
        EasyMock.expect(anthropicMedia.getType("text/plain")).andReturn("url").anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        Map<String, Object> responseFormat = new HashMap<>();
        EasyMock.expect(request.getResponseFormat()).andReturn(responseFormat).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(3.0D).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(2.0).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.getCreated()).andReturn(1L).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();
        Map<String, Object> thinking = new HashMap<>();
        EasyMock.expect(request.getThinking()).andReturn(thinking).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMediaContext()).andReturn(Arrays.asList(mediaContext)).anyTimes();
        EasyMock.expect(request.getAnthropicMedia()).andReturn(anthropicMedia).anyTimes();
        EasyMock.expect(request.getMimeType()).andReturn("text/plain").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("query").anyTimes();

        // 模拟非内联媒体 (如 URL)
        EasyMock.expect(mediaContext.getType()).andReturn("text/plain").anyTimes();
        EasyMock.expect(mediaContext.getData()).andReturn("http://url").anyTimes();

        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();

        EasyMock.replay(request, message, mediaContext, anthropicMedia);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        List<Map<String, Object>> content = (List<Map<String, Object>>) anthropicMessage.getMessages().get(0).getContent();
        Assertions.assertEquals("url", content.get(1).get("type"));
        Map<String, Object> urlMap = (Map<String, Object>) content.get(1).get("source");
        Assertions.assertEquals("http://url", urlMap.get("url"));
        Assertions.assertEquals(2D, anthropicMessage.getTopP());
        Assertions.assertEquals(responseFormat, anthropicMessage.getResponseFormat());
        Assertions.assertEquals(thinking, anthropicMessage.getThinking());
        Assertions.assertEquals(3D, anthropicMessage.getTemperature());
        Assertions.assertEquals(false, anthropicMessage.getStream());
        EasyMock.verify(request, message, mediaContext, anthropicMedia);
    }

    @Test
    public void testAnthropicMessageWithFunCallData() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(1L).anyTimes();
        LLMFunCallData funCallData = EasyMock.createMock(LLMFunCallData.class);
        LLMFunCallRequest funCallRequest = EasyMock.createMock(LLMFunCallRequest.class);
        EasyMock.expect(funCallRequest.getMetadata()).andReturn(Collections.emptyMap()).anyTimes();
        EasyMock.expect(funCallRequest.getReason()).andReturn("reason").anyTimes();
        EasyMock.expect(funCallRequest.getRefer()).andReturn(null).anyTimes();
        LLMFunCallResponse funCallResponse = EasyMock.createMock(LLMFunCallResponse.class);
        EasyMock.expect(funCallResponse.getId()).andReturn("ABC").anyTimes();
        EasyMock.expect(funCallResponse.getMetadata()).andReturn(Collections.emptyMap()).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn(JsonUtils.write(new Object[]{ImmutableMap.of("type", "tool_result", "content", "tool result1")})).anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        // 模拟函数调用数据
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(true).anyTimes();
        EasyMock.expect(request.getFunCallData()).andReturn(funCallData).anyTimes();
        EasyMock.expect(funCallData.getRequests()).andReturn(Arrays.asList(funCallRequest)).anyTimes();
        EasyMock.expect(funCallData.getResponses()).andReturn(Arrays.asList(funCallResponse)).anyTimes();
        EasyMock.expect(funCallRequest.getCreated()).andReturn(1L).anyTimes();
        EasyMock.expect(funCallRequest.getArgs()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(funCallRequest.getName()).andReturn("test_tool").anyTimes();
        EasyMock.expect(funCallRequest.getId()).andReturn("call_1").anyTimes();
        EasyMock.expect(funCallResponse.getName()).andReturn("NAME").anyTimes();
        EasyMock.expect(funCallResponse.getCreated()).andReturn(1L).anyTimes();
        EasyMock.expect(funCallResponse.getResponse()).andReturn(JsonUtils.write(new Object[]{ImmutableMap.of("type", "tool_result", "content", "tool result2")})).anyTimes();

        EasyMock.expect(request.hasFunCall()).andReturn(false).anyTimes();

        EasyMock.replay(request, message, funCallData, funCallRequest, funCallResponse);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(3, anthropicMessage.getMessages().size()); // User Query + Assistant Tool Use + User Tool Result

        // 验证 Assistant 消息 (tool_use)
        AnthropicRouter.AnthropicContent assistantMsg = anthropicMessage.getMessages().get(1);
        Assertions.assertEquals("user", assistantMsg.getRole());
        Object[] assistantContent = (Object[]) assistantMsg.getContent();
        Map<String, Object> toolUse = (Map<String, Object>) assistantContent[0];
        Assertions.assertEquals("tool_result", toolUse.get("type"));

        // 验证 User 消息 (tool_result)
        AnthropicRouter.AnthropicContent userMsg = anthropicMessage.getMessages().get(2);
        Assertions.assertEquals("user", userMsg.getRole());
        Object[] userContent = (Object[]) JsonUtils.read(userMsg.getContent().toString(), Object[].class);
        Map<String, Object> toolResult = (Map<String, Object>) userContent[0];
        Assertions.assertEquals("tool_result", toolResult.get("type"));
        Assertions.assertEquals("tool result1", toolResult.get("content"));

        EasyMock.verify(request, message, funCallData, funCallRequest, funCallResponse);
    }

    @Test
    public void testAnthropicMessageWithTools() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getCreated()).andReturn(1L).anyTimes();
        ProviderFunCall funCall = EasyMock.createMock(ProviderFunCall.class);
        EasyMock.expect(request.getThinking()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMaxTokens()).andReturn(null).anyTimes();
        EasyMock.expect(request.getResponseFormat()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTemperature()).andReturn(null).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(null).anyTimes();
        EasyMock.expect(request.getModel()).andReturn(null).anyTimes();
        EasyMock.expect(request.getTopP()).andReturn(null).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.hasHistory()).andReturn(false).anyTimes();
        EasyMock.expect(request.hasMimeContext()).andReturn(false).anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("hello").anyTimes();
        EasyMock.expect(request.getPrompt()).andReturn(null).anyTimes();
        EasyMock.expect(request.hasFunCallData()).andReturn(false).anyTimes();

        // 模拟工具定义
        EasyMock.expect(request.hasFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(request.getFunCalls()).andReturn(Arrays.asList(funCall)).anyTimes();
        EasyMock.expect(funCall.getProperties()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(funCall.getDescription()).andReturn("tool desc").anyTimes();
        EasyMock.expect(funCall.getName()).andReturn("tool_name").anyTimes();

        EasyMock.replay(request, message, funCall);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(1, anthropicMessage.getTools().size());
        AnthropicRouter.AnthropicTool tool = anthropicMessage.getTools().get(0);
        Assertions.assertEquals("tool_name", tool.getName());
        Assertions.assertEquals("tool desc", tool.getDescription());
        Assertions.assertEquals("object", tool.getInput_schema().get("type"));

        EasyMock.verify(request, message, funCall);
    }

    @Test
    public void testAnthropicContentConstructors() throws Exception {
        // 测试文本构造函数
        AnthropicRouter.AnthropicContent content1 = new AnthropicRouter.AnthropicContent("user", "hello", 1L);
        Assertions.assertEquals("user", content1.getRole());
        Assertions.assertEquals("hello", content1.getContent());

        // 测试 History 构造函数
        History history = EasyMock.createMock(History.class);
        EasyMock.expect(history.getCreated()).andReturn(1L).anyTimes();
        EasyMock.expect(history.isRole(History.ROLE_USER)).andReturn(false).anyTimes();
        EasyMock.expect(history.getContent()).andReturn("assistant reply").anyTimes();
        EasyMock.expect(history.getRole()).andReturn(History.ROLE_ASSISTANT).anyTimes();
        EasyMock.replay(history);

        AnthropicRouter.AnthropicContent content2 = new AnthropicRouter.AnthropicContent(history);
        Assertions.assertEquals("assistant", content2.getRole());
        Assertions.assertEquals("assistant reply", content2.getContent());
        EasyMock.verify(history);

        AnthropicRouter.AnthropicContent content3 = new AnthropicRouter.AnthropicContent(2L);
        Assertions.assertEquals(2L, content3.getCreated());
        Assertions.assertNull(content3.getRole());
        Assertions.assertNull(content3.getContent());
    }

    /**
     * 覆盖构造函数 AnthropicContent(LLMFunCallResponse)：role 为 ROLE_ASSISTANT，content 为单元素数组，元素含 content/tool_use_id/type=TOOL_RESULT
     */
    @Test
    public void testAnthropicContentWithLLMFunCallResponse() throws Exception {
        String responseText = "tool-result-content";
        String toolUseId = "tool-use-123";
        ProviderFunCallResponse llmResponse = ProviderFunCallResponse.builder()
                .response(responseText)
                .id(toolUseId)
                .build();
        AnthropicRouter.AnthropicContent content = new AnthropicRouter.AnthropicContent(llmResponse);
        Assertions.assertEquals(AnthropicRouter.AnthropicContent.ROLE_USER, content.getRole());
        Object[] contentArr = (Object[]) content.getContent();
        Assertions.assertNotNull(contentArr);
        Assertions.assertEquals(1, contentArr.length);
        @SuppressWarnings("unchecked")
        Map<String, Object> response = (Map<String, Object>) contentArr[0];
        Assertions.assertEquals(responseText, response.get("content"));
        Assertions.assertEquals(toolUseId, response.get("tool_use_id"));
        Assertions.assertEquals(AnthropicRouter.AnthropicContent.TOOL_RESULT, response.get("type"));
    }

    @Test
    public void testReader() throws Exception {
        AnthropicRequest request = mockAnthropicRequest();
        LLMConfig config = EasyMock.createMock(LLMConfig.class);
        LLMCallback callback = EasyMock.createMock(LLMCallback.class);
        Message message = EasyMock.createMock(Message.class); // 修复 3: 添加 message mock
        anthropicRouter.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        // 修复 3: 为 anthropicRouter 设置 queue 和 buffer 属性
        anthropicRouter.setQueue(10);
        anthropicRouter.setTimeout(1024);
        anthropicRouter.setDiscard(1025);
        anthropicRouter.setBuffer(1024);
        anthropicRouter.setCapacity(1024);
        anthropicRouter.setQueueTimeout(1024);
        // 修复 3: mock request.getMessage() 和 message.isFromFunCall()
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.isFromFunCall()).andReturn(false).anyTimes();
        // 修复: 添加 getUpstream() 和 getTimeout() 的 mock
        EasyMock.expect(message.getUpstream()).andReturn(null).anyTimes();
        EasyMock.expect(config.getTimeout(EasyMock.anyInt())).andReturn(60000).anyTimes();

        // 测试有图片缓冲区的情况
        EasyMock.expect(config.hasNetworkBuffer()).andReturn(true).anyTimes();
        EasyMock.expect(config.getNetworkBuffer()).andReturn(1024).anyTimes();
        // 修复 4: 确保所有 mock 调用在 replay 之前都已定义
        EasyMock.replay(request, config, callback, message);

        AnthropicReader reader = anthropicRouter.reader(request, config, callback);
        Assertions.assertNotNull(reader);

        // 测试无图片缓冲区的情况
        EasyMock.reset(config);
        EasyMock.expect(config.hasNetworkBuffer()).andReturn(false).anyTimes();
        // 修复: 重置后重新 mock getTimeout
        EasyMock.expect(config.getTimeout(EasyMock.anyInt())).andReturn(60000).anyTimes();
        EasyMock.replay(config);
        reader = anthropicRouter.reader(request, config, callback);
        Assertions.assertNotNull(reader);

        // 修复 4: 确保 verify 被调用
        EasyMock.verify(request, config, callback, message);
    }

    @Test
    public void testInitConfig() throws Exception {
        AnthropicRouter.InitConfig initConfig = new AnthropicRouter.InitConfig();
        initConfig.setUrl("http://init-url");
        AnthropicRouter router = initConfig.anthropicRouter();
        Assertions.assertEquals("http://init-url", router.getUrl());
    }

    /**
     * AnthropicContent.buildJson：args 为 Map 时直接返回该 Map。
     */
    @Test
    public void testAnthropicContentBuildJsonWhenArgsIsMap() throws Exception {
        AnthropicRouter.AnthropicContent content = new AnthropicRouter.AnthropicContent("user", "c", 1L);
        Method m = AnthropicRouter.AnthropicContent.class.getDeclaredMethod("buildJson", Object.class);
        m.setAccessible(true);
        Map<String, Object> input = ImmutableMap.of("k", "v", "a", 1);
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(content, input);
        Assertions.assertSame(input, result);
        Assertions.assertEquals("v", result.get("k"));
        Assertions.assertEquals(1, result.get("a"));
    }

    /**
     * AnthropicContent.buildJson：args 非 Map 但 JsonUtils.transfer 可转为 Map 时返回转换结果。
     */
    @Test
    public void testAnthropicContentBuildJsonWhenArgsTransferToMap() throws Exception {
        AnthropicRouter.AnthropicContent content = new AnthropicRouter.AnthropicContent("user", "c", 1L);
        Method m = AnthropicRouter.AnthropicContent.class.getDeclaredMethod("buildJson", Object.class);
        m.setAccessible(true);
        String jsonLike = "{\"x\":\"y\",\"n\":2}";
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(content, jsonLike);
        Assertions.assertNotNull(result);
        Assertions.assertEquals("y", result.get("x"));
        Assertions.assertEquals(2, result.get("n"));
    }

    /**
     * AnthropicContent.buildJson：转换抛异常时兜底返回 ImmutableMap.of("args", JsonUtils.write(args))。
     */
    @Test
    public void testAnthropicContentBuildJsonWhenTransferFailsFallback() throws Exception {
        AnthropicRouter.AnthropicContent content = new AnthropicRouter.AnthropicContent("user", "c", 1L);
        Method m = AnthropicRouter.AnthropicContent.class.getDeclaredMethod("buildJson", Object.class);
        m.setAccessible(true);
        byte[] args = new byte[]{1, 2, 3};
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(content, args);
        Assertions.assertNotNull(result);
        Assertions.assertEquals(1, result.size());
        Assertions.assertTrue(result.containsKey("args"));
        Assertions.assertNotNull(result.get("args"));
    }

    // ---------- 按 created 升序排序：旧历史在前，当前 query 在中间或末尾，较新历史在后 ----------

    /**
     * 覆盖 AnthropicMessage：按 created 排序后，早于/等于当前 query 的历史在前，晚于当前 query 的历史在后
     */
    @Test
    public void buildHistories_timestamp_splitsAroundUserQuery() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(1000L);
        nr.setQuery("CURRENT_QUERY");

        History before1 = new History();
        before1.setFunction(History.FUN_CHAT);
        before1.setCreated(100L);
        before1.setContent("MSG_BEFORE_1");
        before1.setRole(History.ROLE_USER);

        History before2 = new History();
        before2.setFunction(History.FUN_CHAT);
        before2.setCreated(1000L);
        before2.setContent("MSG_BEFORE_2");
        before2.setRole(History.ROLE_USER);

        History after1 = new History();
        after1.setFunction(History.FUN_CHAT);
        after1.setCreated(1001L);
        after1.setContent("MSG_AFTER");
        after1.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistories(Arrays.asList(before1, before2, after1));

        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(4, anthropicMessage.getMessages().size());
        Assertions.assertEquals("MSG_BEFORE_1", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("MSG_BEFORE_2", anthropicMessage.getMessages().get(1).getContent());
        Assertions.assertEquals("CURRENT_QUERY", anthropicMessage.getMessages().get(2).getContent());
        Assertions.assertEquals("MSG_AFTER", anthropicMessage.getMessages().get(3).getContent());
    }

    /**
     * 全部晚于当前 query：先当前 query，再接较晚历史
     */
    @Test
    public void buildHistories_allAfterTimestamp_userQueryFirst() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(100L);
        nr.setQuery("Q_NOW");

        History after = new History();
        after.setFunction(History.FUN_CHAT);
        after.setCreated(200L);
        after.setContent("HIST_LATER");
        after.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistories(Collections.singletonList(after));

        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        Assertions.assertEquals("Q_NOW", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("HIST_LATER", anthropicMessage.getMessages().get(1).getContent());
    }

    /**
     * 全部早于等于当前 query：先历史再当前 query
     */
    @Test
    public void buildHistories_allOnOrBeforeTimeline_queryLast() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(500L);
        nr.setQuery("QUERY_END");

        History h = new History();
        h.setFunction(History.FUN_CHAT);
        h.setCreated(400L);
        h.setContent("HIST_OLD");
        h.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistories(Collections.singletonList(h));

        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(message);

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(2, anthropicMessage.getMessages().size());
        Assertions.assertEquals("HIST_OLD", anthropicMessage.getMessages().get(0).getContent());
        Assertions.assertEquals("QUERY_END", anthropicMessage.getMessages().get(1).getContent());
    }

    /**
     * 无历史：仅 buildUserQuery
     */
    @Test
    public void buildHistories_splitPath_noHistory_onlyUserQuery() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        ((NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask()).setQuery("ONLY_USER");

        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(Message.build(llmQuery));

        AnthropicRouter.AnthropicMessage anthropicMessage = new AnthropicRouter.AnthropicMessage(request);
        Assertions.assertEquals(1, anthropicMessage.getMessages().size());
        Assertions.assertEquals("ONLY_USER", anthropicMessage.getMessages().get(0).getContent());
    }

    /**
     * 覆盖 buildHistories 中 Fun Call 分支：异常时 catch 并 warn
     */
    @Test
    public void buildHistories_invalidJson_logsWarn() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(AnthropicRouter.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.WARN);
        try {
            History h = new History();
            h.setFunction(History.FUN_FUNCALL);
            h.setType(History.TYPE_QUERY); // 或 TYPE_ANSWER
            h.setContent("{invalid_json");

            AnthropicRequest request = new AnthropicRequest();
            ai.open.right.workflow.flow.llm.MessageDelegate message = new ai.open.right.workflow.flow.llm.MessageDelegate(ai.open.right.ObjectBuilder.buildLLMQuery());
            message.addHistories(Arrays.asList(h));
            request.setMessage(message);
            request.setPrompt("p");

            // 构造时解析历史记录抛出异常，触发 catch 逻辑并产生 Warn 级别日志
            new AnthropicRouter.AnthropicMessage(request);

            List<String> messages = listAppender.list.stream()
                    .map(ILoggingEvent::getFormattedMessage)
                    .collect(Collectors.toList());
            Assertions.assertTrue(
                    messages.stream().anyMatch(m -> m.contains("Unexpected character") || m.contains("Unrecognized token")),
                    "Should log warning for invalid json");
        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
    }
}
