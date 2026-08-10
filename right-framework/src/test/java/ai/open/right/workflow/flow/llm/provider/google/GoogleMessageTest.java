package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.LLMQueryDelegate;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallData;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.util.*;
import java.util.stream.Collectors;
import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import org.slf4j.LoggerFactory;

public class GoogleMessageTest {

    @Test
    public void test() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        List<Map<String, Object>> safe = new ArrayList<>();
        Map<String, Object> tools = new HashMap<String, Object>();
        req.setThinkingConfig(ImmutableMap.of("think", "low"));
        req.setSafetySettings(safe);
        req.setMaxOutputTokens(10);
        req.setFrequencyPenalty(1.0);
        req.setPresencePenalty(2.0);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setTopP(0.3D);
        req.setToolsConfig(tools);
        req.setTopK(2);
        VertexRouter.GoogleMessage message = new VertexRouter.GoogleMessage(req);
        Assert.assertEquals(safe, message.getSafetySettings());
        Assert.assertEquals(tools, message.getToolConfig());
        String actual = "{\"topK\":2,\"presencePenalty\":2.0,\"thinkingConfig\":{\"think\":\"low\"},\"temperature\":0.3,\"frequencyPenalty\":1.0,\"maxOutputTokens\":10,\"topP\":0.3}";
        Assert.assertEquals(JsonUtils.write(message.getConfigs()), actual);
        Assert.assertEquals(Integer.valueOf(message.getContents().size()), Integer.valueOf(1));
        Assert.assertEquals(message.getInstruction().getPart().getText(), "Prompt");
    }

    @Test
    public void testWithFunCallData() throws Exception {
        GoogleRequest req = new GoogleRequest();
        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        providerFunCallData.addFunCall(ProviderFunCallRequest.builder().name("NAME").build(), ProviderFunCallResponse.builder().name("NAME").response("RESPONSE").build());
        req.setFunCallData(providerFunCallData);
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setMaxOutputTokens(10);
        req.setFrequencyPenalty(1.0);
        req.setPresencePenalty(2.0);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setTopP(0.3D);
        req.setTopK(2);
        VertexRouter.GoogleMessage message = new VertexRouter.GoogleMessage(req);
        Assert.assertEquals(Integer.valueOf(3), Integer.valueOf(message.getContents().size()));
    }

    @Test
    public void testFunCallResponseFollowsMatchingRequest() throws Exception {
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

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.setFunCallData(providerFunCallData);
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(3, message.getContents().size());
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT, message.getContents().get(0).getRole());
        Assert.assertNotNull(message.getContents().get(0).getParts().get(0).getFunctionCall());
        Assert.assertEquals("tool_x", message.getContents().get(0).getParts().get(0).getFunctionCall().get("name"));
        Assert.assertEquals("request_1", message.getContents().get(0).getRequestToolCallId());
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT, message.getContents().get(1).getRole());
        Assert.assertNotNull(message.getContents().get(1).getParts().get(0).getFunctionResponse());
        Assert.assertEquals("tool_x", message.getContents().get(1).getParts().get(0).getFunctionResponse().get("name"));
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_USER, message.getContents().get(2).getRole());
        Assert.assertEquals("CURRENT_QUERY", message.getContents().get(2).getParts().get(0).getText());
    }

    @Test
    public void testFunCallResponseFollowsMatchingRequestWithSameCreated() throws Exception {
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

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.setFunCallData(providerFunCallData);
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(3, message.getContents().size());
        Assert.assertEquals("same_id", message.getContents().get(0).getRequestToolCallId());
        Assert.assertNotNull(message.getContents().get(1).getParts().get(0).getFunctionResponse());
    }

    @Test
    public void testFunCallResponsesStayWithOwnIds() throws Exception {
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

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.setFunCallData(providerFunCallData);
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(5, message.getContents().size());
        Assert.assertEquals("id_a", message.getContents().get(0).getRequestToolCallId());
        Assert.assertNotNull(message.getContents().get(1).getParts().get(0).getFunctionResponse());
        Assert.assertEquals("id_b", message.getContents().get(2).getRequestToolCallId());
        Assert.assertNotNull(message.getContents().get(3).getParts().get(0).getFunctionResponse());
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_USER, message.getContents().get(4).getRole());
    }

    @Test
    public void testWithFunCall() throws Exception {
        GoogleRequest req = new GoogleRequest();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setProperties(Collections.singletonMap("HELLO", "WORLD"));
        llmFunCall.setDescription("DESC");
        llmFunCall.setName("NAME");
        req.setFunCalls(Arrays.asList(llmFunCall));
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.setMaxOutputTokens(10);
        req.setFrequencyPenalty(1.0);
        req.setPresencePenalty(2.0);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setTopP(0.3D);
        req.setTopK(2);
        VertexRouter.GoogleMessage message = new VertexRouter.GoogleMessage(req);
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(message.getTools().length));
    }

    @Test
    public void testMime() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        MediaContext mediaContext = new MediaContext();
        mediaContext.setData("http://1.2.3.com");
        req.setMediaContext(Arrays.asList(mediaContext));
        req.setMaxOutputTokens(10);
        req.setFrequencyPenalty(1.0);
        req.setPresencePenalty(2.0);
        req.setTemperature(0.3D);
        req.setPrompt("Prompt");
        req.setTopP(0.3D);
        req.setTopK(2);
        VertexRouter.GoogleMessage message = new VertexRouter.GoogleMessage(req);
        Assert.assertEquals("http://1.2.3.com", message.getContents().getFirst().getParts().get(1).getFile().getUri());
    }

    /** 覆盖 GoogleMessage.labels（@JsonProperty("labels")）的赋值与 getter；序列化时包含 labels。 */
    @Test
    public void testLabels_getAndSerialization() throws Exception {
        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        Map<String, String> labels = new HashMap<>();
        labels.put("app", "my-app");
        labels.put("client", "host1");
        req.setLabels(labels);
        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertNotNull(message.getLabels());
        Assert.assertEquals("my-app", message.getLabels().get("app"));
        Assert.assertEquals("host1", message.getLabels().get("client"));
        String json = JsonUtils.write(message);
        Assert.assertTrue(json.contains("\"labels\""));
        Assert.assertTrue(json.contains("my-app"));
    }

    /** 覆盖 GoogleMessage 中 Fun Call 分支：else -> TYPE_ANSWER -> GoogleContent(response)。 */
    @Test
    public void historyFunCall_typeAnswer_addsContentWithFunctionResponse() throws Exception {
        ProviderFunCallResponse resp = ProviderFunCallResponse.builder().id("id1").response("result").name("tool_a").build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_ANSWER);
        h.setContent(JsonUtils.write(resp));

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.getMessage().addHistories(Arrays.asList(h));
        req.setPrompt("p");

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(2, message.getContents().size());
        GoogleRouter.GoogleMessage.GoogleContent responseContent = message.getContents().stream()
                .filter(each -> GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT.equals(each.getRole()))
                .filter(each -> each.getParts().get(0).getFunctionResponse() != null)
                .findFirst().orElse(null);
        Assert.assertNotNull(responseContent);
        Assert.assertEquals("tool_a", responseContent.getParts().get(0).getFunctionResponse().get("name"));
    }

    /** 覆盖 GoogleMessage 中 Fun Call 分支：else -> TYPE_QUERY -> GoogleContent(request)。 */
    @Test
    public void historyFunCall_typeQuery_addsContentWithFunctionCall() throws Exception {
        ProviderFunCallRequest reqObj = ProviderFunCallRequest.builder().id("id1").name("tool_b").args(ImmutableMap.of("k", "v")).build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_QUERY);
        h.setContent(JsonUtils.write(reqObj));

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.getMessage().addHistories(Arrays.asList(h));
        req.setPrompt("p");

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(2, message.getContents().size());
        GoogleRouter.GoogleMessage.GoogleContent first = message.getContents().get(0);
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT, first.getRole());
        Assert.assertNotNull(first.getParts().get(0).getFunctionCall());
        Assert.assertEquals("tool_b", first.getParts().get(0).getFunctionCall().get("name"));
    }

    /** 覆盖 Fun Call 分支：一条 TYPE_ANSWER + 一条 TYPE_QUERY，顺序为 ANSWER 先、QUERY 后。 */
    @Test
    public void historyFunCall_bothAnswerAndQuery_addsToolResultThenToolUse() throws Exception {
        ProviderFunCallResponse resp = ProviderFunCallResponse.builder().id("id1").response("ok").name("tool_x").build();
        ProviderFunCallRequest reqObj = ProviderFunCallRequest.builder().id("id1").name("tool_x").args(new HashMap<>()).build();
        History hAnswer = new History();
        hAnswer.setFunction(History.FUN_FUNCALL);
        hAnswer.setType(History.TYPE_ANSWER);
        hAnswer.setContent(JsonUtils.write(resp));
        History hQuery = new History();
        hQuery.setFunction(History.FUN_FUNCALL);
        hQuery.setType(History.TYPE_QUERY);
        hQuery.setContent(JsonUtils.write(reqObj));

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.getMessage().addHistories(Arrays.asList(hAnswer, hQuery));
        req.setPrompt("p");

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(3, message.getContents().size());
        GoogleRouter.GoogleMessage.GoogleContent responseContent = message.getContents().stream()
                .filter(each -> GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT.equals(each.getRole()))
                .filter(each -> each.getParts().get(0).getFunctionResponse() != null)
                .findFirst().orElse(null);
        GoogleRouter.GoogleMessage.GoogleContent requestContent = message.getContents().stream()
                .filter(each -> GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT.equals(each.getRole()))
                .filter(each -> each.getParts().get(0).getFunctionCall() != null)
                .findFirst().orElse(null);
        Assert.assertNotNull(responseContent);
        Assert.assertNotNull(requestContent);
    }

    /** appendQuery：当首条 history 的 role 为 ROLE_USER 时，在 contents 前插入一条空 user 轮（满足 Gemini function call 必须紧跟 user/function response 的要求）。 */
    @Test
    public void appendQuery_whenFirstHistoryIsUser_addsEmptyUserContent() throws Exception {
        History h = new History();
        h.setFunction(History.FUN_CHAT);
        h.setRole(History.ROLE_USER);
        h.setContent("user said something");

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.getMessage().addHistories(Arrays.asList(h));
        req.setPrompt("p");

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertTrue(message.getContents().size() >= 2);
        GoogleRouter.GoogleMessage.GoogleContent first = message.getContents().get(0);
        Assert.assertEquals(GoogleRouter.GoogleMessage.GoogleContent.ROLE_USER, first.getRole());
        Assert.assertNotNull(first.getParts());
        Assert.assertEquals(1, first.getParts().size());
        Assert.assertEquals("", first.getParts().get(0).getText());
    }

    /** appendQuery：当首条 history 的 role 非 USER（如 ROLE_ASSISTANT）时，不插入空 user 轮。 */
    @Test
    public void appendQuery_whenFirstHistoryIsAssistant_doesNotAddEmptyUserContent() throws Exception {
        ProviderFunCallResponse resp = ProviderFunCallResponse.builder().id("id1").response("r").name("tool_a").build();
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setType(History.TYPE_ANSWER);
        h.setRole(History.ROLE_ASSISTANT);
        h.setContent(JsonUtils.write(resp));

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        req.getMessage().addHistories(Arrays.asList(h));
        req.setPrompt("p");

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(2, message.getContents().size());
        GoogleRouter.GoogleMessage.GoogleContent responseContent = message.getContents().stream()
                .filter(each -> GoogleRouter.GoogleMessage.GoogleContent.ROLE_ASSISTANT.equals(each.getRole()))
                .filter(each -> each.getParts().get(0).getFunctionResponse() != null)
                .findFirst().orElse(null);
        Assert.assertNotNull(responseContent);
    }

    @Test
    public void buildHistories_usesReasonAsThoughtSignatureOnlyWhenPrintReasonIsEnabled() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nettyRequest = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nettyRequest.setCreated(100L);
        nettyRequest.setQuery("CURRENT_QUERY");

        History history = new History();
        history.setFunction(History.FUN_CHAT);
        history.setCreated(1L);
        history.setRole(History.ROLE_ASSISTANT);
        history.setContent("previous answer");
        history.setReason("thought-signature");

        GoogleRequest withReason = new GoogleRequest();
        withReason.setMessage(Message.build(llmQuery));
        withReason.getMessage().addHistory(history);
        withReason.setPrintReason(true);
        GoogleRouter.GoogleMessage messageWithReason = new GoogleRouter.GoogleMessage(withReason);
        Assert.assertEquals("thought-signature", messageWithReason.getContents().getFirst().getParts().getFirst().getThoughtSignature());

        GoogleRequest withoutReason = new GoogleRequest();
        withoutReason.setMessage(Message.build(llmQuery));
        withoutReason.getMessage().addHistory(history);
        withoutReason.setPrintReason(false);
        GoogleRouter.GoogleMessage messageWithoutReason = new GoogleRouter.GoogleMessage(withoutReason);
        Assert.assertNull(messageWithoutReason.getContents().getFirst().getParts().getFirst().getThoughtSignature());
    }

    // ---------- 按 created 升序排序：旧历史在前，当前 query 在中间或末尾，较新历史在后 ----------

    /**
     * 覆盖 GoogleMessage：按 created 排序后，早于/等于当前 query 的历史在前，晚于当前 query 的历史在后。
     * 首条历史为 ASSISTANT，避免 appendQuery 插入空 user，便于断言顺序。
     */
    @Test
    public void buildHistories_timestamp_splitsAroundUserQuery() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(1000L);
        nr.setQuery("G_CURRENT");

        History before1 = new History();
        before1.setFunction(History.FUN_CHAT);
        before1.setCreated(100L);
        before1.setContent("G_OLD");
        before1.setRole(History.ROLE_ASSISTANT);

        History before2 = new History();
        before2.setFunction(History.FUN_CHAT);
        before2.setCreated(1000L);
        before2.setContent("G_EQ");
        before2.setRole(History.ROLE_ASSISTANT);

        History after1 = new History();
        after1.setFunction(History.FUN_CHAT);
        after1.setCreated(1001L);
        after1.setContent("G_NEW");
        after1.setRole(History.ROLE_USER);

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.getMessage().addHistories(Arrays.asList(before1, before2, after1));
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(4, message.getContents().size());
        Assert.assertEquals("G_OLD", message.getContents().get(0).getParts().get(0).getText());
        Assert.assertEquals("G_EQ", message.getContents().get(1).getParts().get(0).getText());
        Assert.assertEquals("G_CURRENT", message.getContents().get(2).getParts().get(0).getText());
        Assert.assertEquals("G_NEW", message.getContents().get(3).getParts().get(0).getText());
    }

    /** 全部晚于当前 query：先当前 user query，再接较晚历史 */
    @Test
    public void buildHistories_allAfterTimestamp_userQueryFirst() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(100L);
        nr.setQuery("G_Q");

        History after = new History();
        after.setFunction(History.FUN_CHAT);
        after.setCreated(200L);
        after.setContent("G_LATE");
        after.setRole(History.ROLE_ASSISTANT);

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.getMessage().addHistories(Collections.singletonList(after));
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(2, message.getContents().size());
        Assert.assertEquals("G_Q", message.getContents().get(0).getParts().get(0).getText());
        Assert.assertEquals("G_LATE", message.getContents().get(1).getParts().get(0).getText());
    }

    /** 全部早于等于当前 query：兼容空 user 在前，再历史，最后当前 query */
    @Test
    public void buildHistories_allOnOrBeforeTimeline_queryLast() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        NettyRequest nr = (NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask();
        nr.setCreated(500L);
        nr.setQuery("G_END");

        History h = new History();
        h.setFunction(History.FUN_CHAT);
        h.setCreated(400L);
        h.setContent("G_EARLY");
        h.setRole(History.ROLE_USER);

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.getMessage().addHistories(Collections.singletonList(h));
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(3, message.getContents().size());
        Assert.assertEquals("", message.getContents().get(0).getParts().get(0).getText());
        Assert.assertEquals("G_EARLY", message.getContents().get(1).getParts().get(0).getText());
        Assert.assertEquals("G_END", message.getContents().get(2).getParts().get(0).getText());
    }

    /** 无历史：仅 buildUserQuery */
    @Test
    public void buildHistories_noHistory_onlyUserQueryPath() throws Exception {
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        ((NettyRequest) ((LLMQueryDelegate) llmQuery).getWorkTask()).setQuery("G_ONLY");

        GoogleRequest req = new GoogleRequest();
        req.setMessage(Message.build(llmQuery));
        req.setPrompt(null);

        GoogleRouter.GoogleMessage message = new GoogleRouter.GoogleMessage(req);
        Assert.assertEquals(1, message.getContents().size());
        Assert.assertEquals("G_ONLY", message.getContents().get(0).getParts().get(0).getText());
    }

    /** 覆盖 GoogleMessage 中 Fun Call 分支：异常时 catch 并 warn */
    @Test
    public void historyFunCall_invalidJson_logsWarn() throws Exception {
        Logger logger = (Logger) LoggerFactory.getLogger(GoogleRouter.class);
        ListAppender<ILoggingEvent> listAppender = new ListAppender<>();
        listAppender.start();
        logger.addAppender(listAppender);
        Level oldLevel = logger.getLevel();
        logger.setLevel(Level.WARN);
        try {
            History h = new History();
            h.setFunction(History.FUN_FUNCALL);
            h.setType(History.TYPE_ANSWER);
            h.setContent("{invalid_json");

            GoogleRequest req = new GoogleRequest();
            req.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
            req.getMessage().addHistories(Arrays.asList(h));
            req.setPrompt("p");

            new GoogleRouter.GoogleMessage(req);

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
}
