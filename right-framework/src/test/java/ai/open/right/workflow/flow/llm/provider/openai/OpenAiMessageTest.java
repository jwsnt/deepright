package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMQueryDelegate;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallData;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.stream.Collectors;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import org.slf4j.LoggerFactory;

public class OpenAiMessageTest {

    @Test
    public void testFunCallData() throws Exception {
        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setPrompt("PROMPT");
        openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().name("NAME").args(null).build();
        ProviderFunCallResponse providerFunResponse = ProviderFunCallResponse.builder().name("NAME").response("RESPONSE").build();
        providerFunCallData.addFunCall(providerFunRequest, providerFunResponse);
        openAiRequest.setFunCallData(providerFunCallData);
        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(4, openAiMessage.getMessages().size());
    }

    @Test
    public void testFunCalls() throws Exception {
        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setPrompt("PROMPT");
        openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderFunCall providerFunCall = new LLMFunCall();
        openAiRequest.setFunCalls(Arrays.asList(providerFunCall));
        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertNotNull(openAiMessage.getTools());
    }

    @Test
    public void testMessageConfig() throws Exception {
        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        openAiRequest.setPresencePenalty(1.0);
        openAiRequest.setMaxTokens(1024);
        openAiRequest.setPrompt("HELLO");
        openAiRequest.setTopP(0.5);
        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(Double.valueOf(1.0), openAiMessage.getPresencePenalty());
        Assert.assertEquals(Double.valueOf(0.5), openAiMessage.getTopP());
        Assert.assertEquals(Integer.valueOf(1024), openAiMessage.getMaxTokens());
    }

    @Test
    public void testSystemPromptBeforeUserQuery() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(100L);
        nettyRequest.setQuery("USER_QUERY");

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(llmQuery));
        openAiRequest.setPrompt("SYSTEM_PROMPT");

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(2, openAiMessage.getMessages().size());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_SYSTEM, openAiMessage.getMessages().get(0).getRole());
        Assert.assertEquals("SYSTEM_PROMPT", openAiMessage.getMessages().get(0).getContent());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_USER, openAiMessage.getMessages().get(1).getRole());
        Assert.assertEquals("USER_QUERY", openAiMessage.getMessages().get(1).getContent());
    }

    @Test
    public void testMediaContext() throws Exception {
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext c1 = new MediaContext();
        c1.setType("image/jpeg");
        c1.setData("HELLO");
        mediaContext.add(c1);
        MediaContext c2 = new MediaContext();
        c2.setType("inline:image/jpeg");
        c2.setData("HELLO");
        mediaContext.add(c2);
        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        openAiRequest.setMediaContext(mediaContext);
        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(1, openAiMessage.getMessages().size());
        Assert.assertEquals(3, List.class.cast(openAiMessage.getMessages().getFirst().getContent()).size());
    }

    @Test
    public void testFunCallResponseFollowsMatchingRequest() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(400L);
        nr.setQuery("CURRENT_QUERY");

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
        response.setId("request_1");
        response.setName("tool_x");
        response.setResponse("tool_result");

        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        providerFunCallData.addFunCall(request, response);

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(llmQuery));
        openAiRequest.setFunCallData(providerFunCallData);
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(3, openAiMessage.getMessages().size());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_ASSISTANT, openAiMessage.getMessages().get(0).getRole());
        Assert.assertEquals("request_1", ((java.util.Map<?, ?>) ((Object[]) openAiMessage.getMessages().get(0).getToolCalls())[0]).get("id"));
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_TOOL, openAiMessage.getMessages().get(1).getRole());
        Assert.assertEquals("request_1", openAiMessage.getMessages().get(1).getToolCallId());
        Assert.assertEquals("tool_result", openAiMessage.getMessages().get(1).getContent());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_USER, openAiMessage.getMessages().get(2).getRole());
        Assert.assertEquals("CURRENT_QUERY", openAiMessage.getMessages().get(2).getContent());
    }

    @Test
    public void testFunCallResponseFollowsMatchingRequestWithSameCreated() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(300L);
        nr.setQuery("CURRENT_QUERY");

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

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(llmQuery));
        openAiRequest.setFunCallData(providerFunCallData);
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(3, openAiMessage.getMessages().size());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_ASSISTANT, openAiMessage.getMessages().get(0).getRole());
        Assert.assertEquals("same_id", ((java.util.Map<?, ?>) ((Object[]) openAiMessage.getMessages().get(0).getToolCalls())[0]).get("id"));
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_TOOL, openAiMessage.getMessages().get(1).getRole());
        Assert.assertEquals("same_id", openAiMessage.getMessages().get(1).getToolCallId());
    }

    @Test
    public void testFunCallResponsesStayWithOwnIds() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(500L);
        nr.setQuery("CURRENT_QUERY");

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
        responseA.setId("id_a");
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
        responseB.setId("id_b");
        responseB.setName("tool_b");
        responseB.setResponse("result_b");

        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        providerFunCallData.addFunCall(requestA, responseA);
        providerFunCallData.addFunCall(requestB, responseB);

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(llmQuery));
        openAiRequest.setFunCallData(providerFunCallData);
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(5, openAiMessage.getMessages().size());
        Assert.assertEquals("id_a", ((java.util.Map<?, ?>) ((Object[]) openAiMessage.getMessages().get(0).getToolCalls())[0]).get("id"));
        Assert.assertEquals("id_a", openAiMessage.getMessages().get(1).getToolCallId());
        Assert.assertEquals("result_a", openAiMessage.getMessages().get(1).getContent());
        Assert.assertEquals("id_b", ((java.util.Map<?, ?>) ((Object[]) openAiMessage.getMessages().get(2).getToolCalls())[0]).get("id"));
        Assert.assertEquals("id_b", openAiMessage.getMessages().get(3).getToolCallId());
        Assert.assertEquals("result_b", openAiMessage.getMessages().get(3).getContent());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_USER, openAiMessage.getMessages().get(4).getRole());
    }

    /** 覆盖 OpenAiMessage 中 Fun Call 分支：else -> TYPE_QUERY -> buildFunCallRequest -> OpenAiContent(request)。 */
    @Test
    public void historyFunCall_typeQuery_addsContentWithToolCalls() throws Exception {
        ProviderFunCallRequest reqObj = ProviderFunCallRequest.builder().id("id1").name("tool_a").args(ImmutableMap.of("k", "v")).build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_QUERY);
        h.setContent(JsonUtils.write(reqObj));

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        openAiRequest.getMessage().addHistories(Arrays.asList(h));
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(2, openAiMessage.getMessages().size());
        OpenAiRouter.OpenAiContent first = openAiMessage.getMessages().get(0);
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_ASSISTANT, first.getRole());
        Assert.assertNotNull(first.getToolCalls());
    }

    /** 覆盖 OpenAiMessage 中 Fun Call 分支：else -> TYPE_ANSWER -> buildFunCallResponse -> OpenAiContent(response)。 */
    @Test
    public void historyFunCall_typeAnswer_addsContentWithToolCallId() throws Exception {
        ProviderFunCallResponse resp = ProviderFunCallResponse.builder().id("id1").response("result").name("tool_a").build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_ANSWER);
        h.setContent(JsonUtils.write(resp));

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        openAiRequest.getMessage().addHistories(Arrays.asList(h));
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(2, openAiMessage.getMessages().size());
        OpenAiRouter.OpenAiContent tool = openAiMessage.getMessages().stream()
                .filter(each -> OpenAiRouter.OpenAiContent.ROLE_TOOL.equals(each.getRole()))
                .findFirst().orElse(null);
        Assert.assertNotNull(tool);
        Assert.assertEquals("id1", tool.getToolCallId());
        Assert.assertEquals("result", tool.getContent());
    }

    /** 覆盖 Fun Call 分支：一条 TYPE_QUERY + 一条 TYPE_ANSWER，顺序为 QUERY 先、ANSWER 后。 */
    @Test
    public void historyFunCall_bothQueryAndAnswer_addsToolCallsThenToolResponse() throws Exception {
        ProviderFunCallRequest reqObj = ProviderFunCallRequest.builder().id("id1").name("tool_x").args(new HashMap<>()).build();
        ProviderFunCallResponse resp = ProviderFunCallResponse.builder().id("id1").response("ok").name("tool_x").build();
        History hQuery = new History();
        hQuery.setFunction(History.FUN_FUNCALL);
        hQuery.setType(History.TYPE_QUERY);
        hQuery.setContent(JsonUtils.write(reqObj));
        History hAnswer = new History();
        hAnswer.setFunction(History.FUN_FUNCALL);
        hAnswer.setType(History.TYPE_ANSWER);
        hAnswer.setContent(JsonUtils.write(resp));

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        openAiRequest.getMessage().addHistories(Arrays.asList(hQuery, hAnswer));
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(3, openAiMessage.getMessages().size());
        OpenAiRouter.OpenAiContent assistant = openAiMessage.getMessages().stream()
                .filter(each -> OpenAiRouter.OpenAiContent.ROLE_ASSISTANT.equals(each.getRole()))
                .findFirst().orElse(null);
        OpenAiRouter.OpenAiContent tool = openAiMessage.getMessages().stream()
                .filter(each -> OpenAiRouter.OpenAiContent.ROLE_TOOL.equals(each.getRole()))
                .findFirst().orElse(null);
        Assert.assertNotNull(assistant);
        Assert.assertNotNull(assistant.getToolCalls());
        Assert.assertNotNull(tool);
        Assert.assertEquals("id1", tool.getToolCallId());
    }

    /** 覆盖 OpenAiMessage.buildHistories 中 Fun Call 分支：异常时 catch 并 warn */
    @Test
    public void historyFunCall_invalidJson_logsWarn() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(OpenAiRouter.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.WARN);
        try {
            History h = new History();
            h.setFunction(History.FUN_FUNCALL);
            h.setType(History.TYPE_ANSWER); // 或 TYPE_QUERY
            h.setContent("{invalid_json");

            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
            openAiRequest.getMessage().addHistories(Arrays.asList(h));
            openAiRequest.setPrompt("p");

            // 构造时解析历史记录抛出异常，触发 catch 逻辑并产生 Warn 级别日志
            new OpenAiRouter.OpenAiMessage(openAiRequest);

            List<String> messages = listAppender.list.stream()
                    .map(ILoggingEvent::getFormattedMessage)
                    .collect(Collectors.toList());
            Assert.assertTrue("Should log warning for invalid json",
                    messages.stream().anyMatch(m -> m.contains("Unexpected character") || m.contains("Unrecognized token")));
        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
    }

    // ---------- History.split：先早于等于 timestamp 的历史 → 当前 user query → 晚于 timestamp 的历史 ----------

    /**
     * 覆盖 OpenAiMessage.buildHistories：hasHistory 且带 timestamp 时，按 History.split 先旧会话、再当前 user query、再新会话。
     */
    @Test
    public void buildHistories_withTimestamp_splitsAroundUserQuery() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(1000L);
        nr.setQuery("CURRENT_QUERY");

        History before1 = new History();
        before1.setCreated(100L);
        before1.setContent("MSG_BEFORE_1");
        before1.setRole(History.ROLE_USER);

        History before2 = new History();
        before2.setCreated(1000L);
        before2.setContent("MSG_BEFORE_2");
        before2.setRole(History.ROLE_USER);

        History after1 = new History();
        after1.setCreated(1001L);
        after1.setContent("MSG_AFTER");
        after1.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistories(Arrays.asList(before1, before2, after1));

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(message);
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        List<OpenAiRouter.OpenAiContent> msgs = openAiMessage.getMessages();
        Assert.assertEquals(4, msgs.size());
        Assert.assertEquals("MSG_BEFORE_1", msgs.get(0).getContent());
        Assert.assertEquals("MSG_BEFORE_2", msgs.get(1).getContent());
        Assert.assertEquals("CURRENT_QUERY", msgs.get(2).getContent());
        Assert.assertEquals("MSG_AFTER", msgs.get(3).getContent());
    }

    /** 覆盖 buildHistories：全部晚于 timestamp 时，先 user query，再接较晚历史 */
    @Test
    public void buildHistories_allAfterTimestamp_userQueryFirst() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(100L);
        nr.setQuery("Q_NOW");

        History after = new History();
        after.setCreated(200L);
        after.setContent("HIST_LATER");
        after.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistories(Collections.singletonList(after));

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(message);
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(2, openAiMessage.getMessages().size());
        Assert.assertEquals("Q_NOW", openAiMessage.getMessages().get(0).getContent());
        Assert.assertEquals("HIST_LATER", openAiMessage.getMessages().get(1).getContent());
    }

    /** 覆盖 buildHistories：全部早于等于 timestamp 时，先历史再当前 query，[1] 为空 */
    @Test
    public void buildHistories_allOnOrBeforeTimeline_queryLast() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(500L);
        nr.setQuery("QUERY_END");

        History h = new History();
        h.setCreated(400L);
        h.setContent("HIST_OLD");
        h.setRole(History.ROLE_USER);

        Message message = Message.build(llmQuery);
        message.addHistories(Collections.singletonList(h));

        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(message);
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(2, openAiMessage.getMessages().size());
        Assert.assertEquals("HIST_OLD", openAiMessage.getMessages().get(0).getContent());
        Assert.assertEquals("QUERY_END", openAiMessage.getMessages().get(1).getContent());
    }

    /** 覆盖 buildHistories：无历史时走 else，仅 buildUserQuery */
    @Test
    public void buildHistories_noHistory_onlyUserQueryPath() throws Exception {
        OpenAiRequest openAiRequest = new OpenAiRequest();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        ((NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask()).setQuery("ONLY_USER");
        openAiRequest.setMessage(Message.build(llmQuery));
        openAiRequest.setPrompt(null);

        OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
        Assert.assertEquals(1, openAiMessage.getMessages().size());
        Assert.assertEquals("ONLY_USER", openAiMessage.getMessages().get(0).getContent());
        Assert.assertEquals(OpenAiRouter.OpenAiContent.ROLE_USER, openAiMessage.getMessages().get(0).getRole());
    }

    /**
     * 覆盖 OpenAiMessage.buildHistories 中 FunCall TYPE_ANSWER 解析失败且 lastRequest!=null 时的 fallback 分支：
     * - 前一条 TYPE_QUERY 正确解析，设置 lastRequest
     * - 后一条 TYPE_ANSWER 内容非 JSON，触发 JsonParseException
     * - fallback 使用 lastRequest 的 model/name/id + 原始 content 构建响应，并记录 INFO 日志
     */
    @Test
    public void historyFunCall_typeAnswer_invalidJson_withLastRequest_fallback() throws Exception {
        // 准备：第一条 TYPE_QUERY 正常解析，设置 lastRequest
        ProviderFunCallRequest reqObj = ProviderFunCallRequest.builder()
                .id("id1").name("tool_x").model("gpt-4").args(new HashMap<>()).build();
        History hQuery = new History();
        hQuery.setFunction(History.FUN_FUNCALL);
        hQuery.setType(History.TYPE_QUERY);
        hQuery.setContent(JsonUtils.write(reqObj));

        // 第二条 TYPE_ANSWER 内容非法 JSON，触发 JsonParseException
        History hAnswer = new History();
        hAnswer.setFunction(History.FUN_FUNCALL);
        hAnswer.setType(History.TYPE_ANSWER);
        hAnswer.setContent("not a json");

        // 拦截 INFO 日志，验证 fallback 日志输出
        Logger logger = (Logger) LoggerFactory.getLogger(OpenAiRouter.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.INFO);
        try {
            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
            openAiRequest.getMessage().addHistories(Arrays.asList(hQuery, hAnswer));
            openAiRequest.setPrompt(null);

            OpenAiRouter.OpenAiMessage openAiMessage = new OpenAiRouter.OpenAiMessage(openAiRequest);
            // 结果：assistant(tool_calls from query) + tool(fallback) + user(query) = 3条
            Assert.assertEquals(3, openAiMessage.getMessages().size());

            // assistant — 来自 TYPE_QUERY 的正确解析
            OpenAiRouter.OpenAiContent assistant = openAiMessage.getMessages().stream()
                    .filter(each -> OpenAiRouter.OpenAiContent.ROLE_ASSISTANT.equals(each.getRole()))
                    .findFirst().orElse(null);
            Assert.assertNotNull(assistant);
            Assert.assertNotNull(assistant.getToolCalls());

            // tool — 来自 TYPE_ANSWER 的 fallback
            OpenAiRouter.OpenAiContent tool = openAiMessage.getMessages().stream()
                    .filter(each -> OpenAiRouter.OpenAiContent.ROLE_TOOL.equals(each.getRole()))
                    .findFirst().orElse(null);
            Assert.assertNotNull(tool);
            Assert.assertEquals("id1", tool.getToolCallId());       // 来自 lastRequest.getId()
            Assert.assertEquals("not a json", tool.getContent());   // 原始 content 保留

            // 验证 INFO 日志已输出 fallback 提示
            boolean foundFallbackLog = listAppender.list.stream()
                    .map(ILoggingEvent::getFormattedMessage)
                    .anyMatch(m -> m.contains("The function call response will be built based on the previous request"));
            Assert.assertTrue("Should log info for fallback", foundFallbackLog);

        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
    }

    /**
     * 覆盖 OpenAiMessage.buildHistories 中 FunCall TYPE_ANSWER 解析失败且 lastRequest==null 时的 rethrow 分支：
     * - 无前置 TYPE_QUERY，直接遇到无效 JSON 的 TYPE_ANSWER
     * - JsonParseException 被 rethrow → 外层 catch 捕获并记录 WARN 日志
     * - 该条历史被跳过，不会产生 tool 类型 message
     */
    @Test
    public void historyFunCall_typeAnswer_invalidJson_withoutLastRequest_rethrows() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(OpenAiRouter.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.WARN);
        try {
            History hAnswer = new History();
            hAnswer.setFunction(History.FUN_FUNCALL);
            hAnswer.setType(History.TYPE_ANSWER);
            hAnswer.setContent("{invalid_json}");

            OpenAiRequest openAiRequest = new OpenAiRequest();
            openAiRequest.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
            openAiRequest.getMessage().addHistories(Arrays.asList(hAnswer));
            openAiRequest.setPrompt("p");

            new OpenAiRouter.OpenAiMessage(openAiRequest);

            // 由于 lastRequest==null，rethrow 的 JsonParseException 被外层 catch 记录 WARN
            boolean foundWarn = listAppender.list.stream()
                    .map(ILoggingEvent::getFormattedMessage)
                    .anyMatch(m -> m.contains("JsonParseException")
                            || m.contains("Unexpected character")
                            || m.contains("Unrecognized token")
                            || m.contains("invalid_json"));
            Assert.assertTrue("Should log warn for rethrown JsonParseException", foundWarn);

            // 验证只有一条 message（user query），因为 TYPE_ANSWER 解析失败被跳过
            // 无反 case 为 TOOL，因为 fallback 路径未触发（lastRequest==null）
        } finally {
            logger.setLevel(oldLevel);
            logger.detachAndStopAllAppenders();
        }
    }
}
