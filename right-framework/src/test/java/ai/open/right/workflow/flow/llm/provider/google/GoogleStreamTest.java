package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderData;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.MediaInlineData;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.springframework.util.ResourceUtils;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.*;

public class GoogleStreamTest {

    private TrackFunCallService trackFunCallService;
    private TokenStatistic tokenStatistic;
    private MediaInlineService mediaInlineService;
    private NotifierService notifierService;
    private ProviderReason providerReason;
    private SignalStream signalStream;
    private HistoryStore historyStore;
    private GoogleRequest googleRequest;
    private Message message;

    @BeforeEach
    public void setUp() {
        trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        tokenStatistic = ObjectBuilder.buildTokenStatistic();
        mediaInlineService = ObjectBuilder.buildMediaInlineService();
        notifierService = EasyMock.createMock(NotifierService.class);
        providerReason = ObjectBuilder.getProviderReason();
        signalStream = EasyMock.createMock(SignalStream.class);
        historyStore = ObjectBuilder.buildHistoryStore();
        googleRequest = EasyMock.createMock(GoogleRequest.class);
        message = EasyMock.createMock(Message.class);
    }

    private void prepareMocks() {
        EasyMock.expect(googleRequest.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(googleRequest.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.isFromFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(message.getWorkflow()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("QUERY").anyTimes();
        EasyMock.expect(message.getChat()).andReturn("CHAT").anyTimes();
        EasyMock.expect(message.getBiz()).andReturn("BIZ").anyTimes();
        EasyMock.expect(message.getDevice()).andReturn("DEVICE").anyTimes();
        EasyMock.expect(message.getUserContext()).andReturn(ObjectBuilder.buildEmpty()).anyTimes();
        EasyMock.expect(message.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(message.getCreated()).andReturn(System.currentTimeMillis()).anyTimes();
        EasyMock.expect(message.getNotifier()).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(message.getOriginal()).andReturn("ORIGINAL").anyTimes();
        EasyMock.expect(message.getPrevious()).andReturn("PREVIOUS").anyTimes();
        EasyMock.expect(message.getInitial()).andReturn("INITIAL").anyTimes();
        EasyMock.expect(message.getDeepness()).andReturn(0).anyTimes();
        EasyMock.expect(message.getConversation()).andReturn("CONVERSATION").anyTimes();
        EasyMock.expect(message.getUpstream()).andReturn("UPSTREAM").anyTimes();
        EasyMock.expect(message.getTrace()).andReturn("TRACE").anyTimes();
        EasyMock.expect(message.getProtocol()).andReturn(Protocol.CHAT).anyTimes();

        EasyMock.expect(googleRequest.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(googleRequest.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(googleRequest.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(googleRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(googleRequest.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(googleRequest.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(googleRequest.getNotifier(EasyMock.anyString())).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(googleRequest.getTokenFirst()).andReturn(100).anyTimes();
        EasyMock.expect(googleRequest.getTokenBuffer()).andReturn(100).anyTimes();
        EasyMock.expect(googleRequest.getContainHistories()).andReturn(false).anyTimes();

        EasyMock.replay(googleRequest, message);
    }

    @org.junit.jupiter.api.Test
    public void testStream() throws Exception {
        prepareMocks();

        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(trackFunCallService)
                .tokenStatistic(tokenStatistic)
                .mediaInlineService(mediaInlineService)
                .notifierService(notifierService)
                .providerReason(providerReason)
                .signalStream(signalStream)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build());

        Map<String, Object> body = new HashMap<>();
        List<Map<String, Object>> candidates = new ArrayList<>();
        Map<String, Object> candidate = new HashMap<>();
        Map<String, Object> content = new HashMap<>();
        List<Map<String, Object>> parts = new ArrayList<>();
        Map<String, Object> part = new HashMap<>();
        part.put("text", "Hello");
        part.put("thoughtSignature", "thinking...");
        parts.add(part);
        content.put("parts", parts);
        content.put("role", "model");
        candidate.put("content", content);
        candidate.put("finishReason", "STOP");
        candidates.add(candidate);
        body.put("candidates", candidates);

        Map<String, Object> usage = new HashMap<>();
        usage.put("totalTokenCount", 10);
        body.put("usageMetadata", usage);

        String source = JsonUtils.write(body);
        Boolean finished = stream.stream(source);

        Assertions.assertTrue(finished);
        Assertions.assertEquals("Hello", stream.getContentBuffer().toString());
        Assertions.assertNull(stream.getReasoning(), "Google thought signatures are only attached to function calls");
    }

    @org.junit.jupiter.api.Test
    public void testStreamWithFunctionCall() throws Exception {
        prepareMocks();

        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(trackFunCallService)
                .tokenStatistic(tokenStatistic)
                .mediaInlineService(mediaInlineService)
                .notifierService(notifierService)
                .providerReason(providerReason)
                .signalStream(signalStream)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build());

        Map<String, Object> body = new HashMap<>();
        List<Map<String, Object>> candidates = new ArrayList<>();
        Map<String, Object> candidate = new HashMap<>();
        Map<String, Object> content = new HashMap<>();
        List<Map<String, Object>> parts = new ArrayList<>();
        Map<String, Object> part = new HashMap<>();
        Map<String, Object> functionCall = new HashMap<>();
        functionCall.put("name", "get_weather");
        functionCall.put("args", new HashMap<>());
        part.put("functionCall", functionCall);
        parts.add(part);
        content.put("parts", parts);
        candidate.put("content", content);
        candidates.add(candidate);
        body.put("candidates", candidates);

        String source = JsonUtils.write(body);
        Boolean finished = stream.stream(source);

        Assertions.assertFalse(finished);
    }

    @org.junit.jupiter.api.Test
    public void testAtonce() throws Exception {
        prepareMocks();

        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(trackFunCallService)
                .tokenStatistic(tokenStatistic)
                .mediaInlineService(mediaInlineService)
                .notifierService(notifierService)
                .providerReason(providerReason)
                .signalStream(signalStream)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build());

        Map<String, Object> body = new HashMap<>();
        List<Map<String, Object>> candidates = new ArrayList<>();
        candidates.add(ImmutableMap.of("A", "B"));
        body.put("candidates", candidates);

        String source = JsonUtils.write(body);
        try {
            stream.atonce(source);
            Assert.fail();
        } catch (WorkflowException e) {
            Assert.assertEquals(e.getCode(), ProtocolCode.C914);
        }
    }

    @Test
    public void testOnceWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

                    @Override
                    public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                        Assert.assertEquals(Integer.valueOf(3630), tokenData.getTotal());
                        Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
                    }

                    @Override
                    public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                        return List.of();
                    }

                    @Override
                    public List<TokenData> readAll(Dimension dimension) throws Exception {
                        return List.of();
                    }

                    @Override
                    public TokenData read(Dimension dimension, String model) throws Exception {
                        return null;
                    }

                    @Override
                    public TokenData read(Dimension dimension) throws Exception {
                        return null;
                    }

                    @Override
                    public Set<String> models() throws Exception {
                        return Set.of();
                    }
                })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("阅读"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.atonce("{\n" +
                "  \"candidates\": [\n" +
                "    {\n" +
                "      \"content\": {\n" +
                "        \"role\": \"model\",\n" +
                "        \"parts\": [\n" +
                "          {\n" +
                "            \"text\": \"阅读\"\n" +
                "          }\n" +
                "        ]\n" +
                "      },\n" +
                "      \"finishReason\": \"STOP\",\n" +
                "      \"safetyRatings\": [\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.09667969,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.12988281\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.22265625,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.15917969\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.14355469,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.09277344\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.20800781,\n" +
                "          \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                "          \"severityScore\": 0.203125\n" +
                "        }\n" +
                "      ],\n" +
                "      \"avgLogprobs\": -0.0027239176410215871\n" +
                "    }\n" +
                "  ],\n" +
                "  \"usageMetadata\": {\n" +
                "    \"promptTokenCount\": 3576,\n" +
                "    \"candidatesTokenCount\": 54,\n" +
                "    \"totalTokenCount\": 3630\n" +
                "  }\n" +
                "}\n"));
        Assert.assertEquals(Integer.valueOf(3630), stream.getSegment().getUsage().getTotal());
        Assert.assertEquals(Integer.valueOf(0), stream.getSegment().getUsage().getCache());
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testOnceWithFinished2() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("阅读"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.atonce("{\n" +
                "  \"candidates\": [\n" +
                "    {\n" +
                "      \"content\": {\n" +
                "        \"role\": \"model\",\n" +
                "        \"parts\": [\n" +
                "          {\n" +
                "            \"text\": \"阅读\\n\"\n" +
                "          }\n" +
                "        ]\n" +
                "      },\n" +
                "      \"finishReason\": \"STOP\",\n" +
                "      \"safetyRatings\": [\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.09667969,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.12988281\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.22265625,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.15917969\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.14355469,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.09277344\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.20800781,\n" +
                "          \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                "          \"severityScore\": 0.203125\n" +
                "        }\n" +
                "      ],\n" +
                "      \"avgLogprobs\": -0.0027239176410215871\n" +
                "    }\n" +
                "  ],\n" +
                "  \"usageMetadata\": {\n" +
                "    \"promptTokenCount\": 3576,\n" +
                "    \"candidatesTokenCount\": 54,\n" +
                "    \"totalTokenCount\": 3630\n" +
                "  }\n" +
                "}\n"));
        EasyMock.verify(s, t, c);
        Assert.assertEquals("阅读\n", stream.getContentBuffer().toString());
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithJsonError() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("I'm now carefully checking the generated image against the original prompt. The image of Pikachu in a forest is a good fit. I'm focusing on the alignment between the image's elements and the user's description. The image seems to capture the essence of what was requested"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        stream.atonce("\n" +
                "  \"candidates\": [\n" +
                "    {\n" +
                "      \"content\": {\n" +
                "        \"role\": \"model\",\n" +
                "        \"parts\": [\n" +
                "          {\n" +
                "            \"text\": \"${I_02;S_00;S_02;S_05=100M流量包}\\n好的，您要购买100M流量包，请稍等，我正在为您准备购买链接。 \\n\"\n" +
                "          }\n" +
                "        ]\n" +
                "      },\n" +
                "      \"finishReason\": \"STOP\",\n" +
                "      \"safetyRatings\": [\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.09667969,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.12988281\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.22265625,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.15917969\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.14355469,\n" +
                "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "          \"severityScore\": 0.09277344\n" +
                "        },\n" +
                "        {\n" +
                "          \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                "          \"probability\": \"NEGLIGIBLE\",\n" +
                "          \"probabilityScore\": 0.20800781,\n" +
                "          \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                "          \"severityScore\": 0.203125\n" +
                "        }\n" +
                "      ],\n" +
                "      \"avgLogprobs\": -0.0027239176410215871\n" +
                "    }\n" +
                "  ],\n" +
                "  \"usageMetadata\": {\n" +
                "    \"promptTokenCount\": 3576,\n" +
                "    \"candidatesTokenCount\": 54,\n" +
                "    \"totalTokenCount\": 3630\n" +
                "  }\n" +
                "}\n");
        Assert.fail();
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithNull() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        try {
            GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                    .trackFunCallService(t)
                    .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                    .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                    .notifierService(r)
                    .providerReason(ObjectBuilder.getProviderReason())
                    .signalStream(s)
                    .historyStore(h)
                    .namesService(ObjectBuilder.buildNamesService())
                    .request(c)
                    .build());
            stream.atonce("\n" +
                    "  \"candidates\": [\n" +
                    "    {\n" +
                    "      \"content\": {\n" +
                    "        \"role\": \"model\",\n" +
                    "        \"parts\": [\n" +
                    "          {\n" +
                    "            \"text\": \"${I_02;S_00;S_02;S_05=100M流量包}\\n好的，您要购买100M流量包，请稍等，我正在为您准备购买链接。 \\n\"\n" +
                    "          }\n" +
                    "        ]\n" +
                    "      },\n" +
                    "      \"finishReason\": \"STOP\",\n" +
                    "      \"safetyRatings\": [\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.09667969,\n" +
                    "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                    "          \"severityScore\": 0.12988281\n" +
                    "        },\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.22265625,\n" +
                    "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                    "          \"severityScore\": 0.15917969\n" +
                    "        },\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.14355469,\n" +
                    "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                    "          \"severityScore\": 0.09277344\n" +
                    "        },\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.20800781,\n" +
                    "          \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                    "          \"severityScore\": 0.203125\n" +
                    "        }\n" +
                    "      ],\n" +
                    "      \"avgLogprobs\": -0.0027239176410215871\n" +
                    "    }\n" +
                    "  ],\n" +
                    "  \"usageMetadata\": {\n" +
                    "    \"promptTokenCount\": 3576,\n" +
                    "    \"candidatesTokenCount\": 54,\n" +
                    "    \"totalTokenCount\": 3630\n" +
                    "  }\n" +
                    "}\n");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test(expected = WorkflowException.class)
    public void testOnceWithFinishedWithEmptyMessage() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        try {
            GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                    .trackFunCallService(t)
                    .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                    .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                    .notifierService(r)
                    .providerReason(ObjectBuilder.getProviderReason())
                    .signalStream(s)
                    .historyStore(h)
                    .namesService(ObjectBuilder.buildNamesService())
                    .request(c)
                    .build());
            stream.atonce("{\n" +
                    "  \"candidates_\": [\n" +
                    "    {\n" +
                    "      \"content\": {\n" +
                    "        \"role\": \"model\",\n" +
                    "        \"parts\": [\n" +
                    "          {\n" +
                    "            \"text\": \"${I_02;S_00;S_02;S_05=100M流量包}\\n好的，您要购买100M流量包，请稍等，我正在为您准备购买链接。 \\n\"\n" +
                    "          }\n" +
                    "        ]\n" +
                    "      },\n" +
                    "      \"finishReason\": \"STOP\",\n" +
                    "      \"safetyRatings\": [\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.09667969,\n" +
                    "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                    "          \"severityScore\": 0.12988281\n" +
                    "        },\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.22265625,\n" +
                    "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                    "          \"severityScore\": 0.15917969\n" +
                    "        },\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.14355469,\n" +
                    "          \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                    "          \"severityScore\": 0.09277344\n" +
                    "        },\n" +
                    "        {\n" +
                    "          \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                    "          \"probability\": \"NEGLIGIBLE\",\n" +
                    "          \"probabilityScore\": 0.20800781,\n" +
                    "          \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                    "          \"severityScore\": 0.203125\n" +
                    "        }\n" +
                    "      ],\n" +
                    "      \"avgLogprobs\": -0.0027239176410215871\n" +
                    "    }\n" +
                    "  ],\n" +
                    "  \"usageMetadata\": {\n" +
                    "    \"promptTokenCount\": 3576,\n" +
                    "    \"candidatesTokenCount\": 54,\n" +
                    "    \"totalTokenCount\": 3630\n" +
                    "  }\n" +
                    "}\n");
            Assert.fail();
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test
    public void testStreamWithNotFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(false).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        String json = "[{\n" +
                "  \"candidates\": [\n" +
                "    {\n" +
                "      \"content\": {\n" +
                "        \"role\": \"model\",\n" +
                "        \"parts\": [\n" +
                "          {\n" +
                "            \"text\": \"${\"\n" +
                "          }\n" +
                "        ]\n" +
                "      }\n" +
                "    }\n" +
                "  ]\n" +
                "}\n" +
                ",\n" +
                "  {\n" +
                "    \"candidates\": [\n" +
                "      {\n" +
                "        \"content\": {\n" +
                "          \"role\": \"model\",\n" +
                "          \"parts\": [\n" +
                "            {\n" +
                "              \"text\": \"I_02;S_00;S_02;S\"\n" +
                "            }\n" +
                "          ]\n" +
                "        },\n" +
                "        \"safetyRatings\": [\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.29492188,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.1796875\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.37890625,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.17480469\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.296875,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.15429688\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.3671875,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.12988281\n" +
                "          }\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  }\n" +
                ",\n" +
                "  {\n" +
                "    \"candidates\": [\n" +
                "      {\n" +
                "        \"content\": {\n" +
                "          \"role\": \"model\",\n" +
                "          \"parts\": [\n" +
                "            {\n" +
                "              \"text\": \"_05=100M流量包}\\n好的，您要\"\n" +
                "            }\n" +
                "          ]\n" +
                "        },\n" +
                "        \"safetyRatings\": [\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.2421875,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.16601563\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.32421875,\n" +
                "            \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                "            \"severityScore\": 0.20605469\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.29101563,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.19824219\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.33398438,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.1875\n" +
                "          }\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  }\n" +
                ",\n" +
                "  {\n" +
                "    \"candidates\": [\n" +
                "      {\n" +
                "        \"content\": {\n" +
                "          \"role\": \"model\",\n" +
                "          \"parts\": [\n" +
                "            {\n" +
                "              \"text\": \"购买100M流量包，请稍等，我正在为您准备购买链接。 \\n\"\n" +
                "            }\n" +
                "          ]\n" +
                "        },\n" +
                "        \"safetyRatings\": [\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_HATE_SPEECH\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.09667969,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.12988281\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_DANGEROUS_CONTENT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.22265625,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.15917969\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_HARASSMENT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.14355469,\n" +
                "            \"severity\": \"HARM_SEVERITY_NEGLIGIBLE\",\n" +
                "            \"severityScore\": 0.09277344\n" +
                "          },\n" +
                "          {\n" +
                "            \"category\": \"HARM_CATEGORY_SEXUALLY_EXPLICIT\",\n" +
                "            \"probability\": \"NEGLIGIBLE\",\n" +
                "            \"probabilityScore\": 0.20800781,\n" +
                "            \"severity\": \"HARM_SEVERITY_LOW\",\n" +
                "            \"severityScore\": 0.203125\n" +
                "          }\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  }\n" +
                ",\n" +
                "  {\n" +
                "    \"candidates\": [\n" +
                "      {\n" +
                "        \"content\": {\n" +
                "          \"role\": \"model\",\n" +
                "          \"parts\": [\n" +
                "            {\n" +
                "              \"text\": \"\"\n" +
                "            }\n" +
                "          ]\n" +
                "        }\n" +
                "      }\n" +
                "    ],\n" +
                "    \"usageMetadata\": {\n" +
                "      \"promptTokenCount\": 3576,\n" +
                "      \"candidatesTokenCount\": 54,\n" +
                "      \"totalTokenCount\": 3630\n" +
                "    }\n" +
                "  }\n" +
                "]";
        List<Object> data = JsonUtils.read(json, List.class);
        boolean finish = false;
        for (Object each : data) {
            finish = finish || stream.stream(JsonUtils.write(each));
        }
        Assert.assertFalse(finish);
        EasyMock.verify(s, t, h, c);
    }

    @Test
    public void testStreamWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

                    @Override
                    public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                        Assert.assertEquals(Integer.valueOf(2800), tokenData.getTotal());
                        Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
                    }

                    @Override
                    public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                        return List.of();
                    }

                    @Override
                    public List<TokenData> readAll(Dimension dimension) throws Exception {
                        return List.of();
                    }

                    @Override
                    public TokenData read(Dimension dimension, String model) throws Exception {
                        return null;
                    }

                    @Override
                    public TokenData read(Dimension dimension) throws Exception {
                        return null;
                    }

                    @Override
                    public Set<String> models() throws Exception {
                        return Set.of();
                    }
                })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {

            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse_stream.json"), StandardCharsets.UTF_8);
        List<Object> data = JsonUtils.read(json, List.class);
        boolean finish = false;
        for (Object each : data) {
            finish = finish || stream.stream(JsonUtils.write(each));
        }
        Assert.assertTrue(finish);
        EasyMock.verify(s, t, c);
    }


    @Test
    public void testStreamWithOutFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {

            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponseWithOutFinish.json"), StandardCharsets.UTF_8);
        List<Object> data = JsonUtils.read(request, List.class);
        boolean finish = false;
        for (Object each : data) {
            finish = finish || stream.stream(JsonUtils.write(each));
        }
        Assert.assertFalse(finish);
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testStreamWithTransmit() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void storeConversation(String content) {

            }
        };
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse_stream.json"), StandardCharsets.UTF_8);
        List<Object> data = JsonUtils.read(request, List.class);
        boolean finish = false;
        for (Object each : data) {
            finish = finish || stream.stream(JsonUtils.write(each));
        }
        // Change To AtOnce
        Assert.assertTrue(finish);
        String expect = "${I_01;S_00;S_01}\n" +
                "我们是科技，主要做非洲业务。\n" +
                "\n" +
                "${S_03=2,3}\n" +
                "我们这里有两种流量包，一个是肯尼亚的5G流量包，一个是肯尼亚的套餐。您比较偏向哪一种呢？\n";
        Assert.assertEquals(expect, stream.getContentBuffer().toString());
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testOnce() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {

            protected void storeConversation(String content) {

            }
        };
        // Change To AtOnce
        Assert.assertTrue(stream.atonce(IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse.json"), StandardCharsets.UTF_8)));
        String expect = "好的，这是一个Python脚本，用于读取你提供的两个文件并打印它们的内容。\n" +
                "\n" +
                "请注意：\n" +
                "1.  **文件路径**：这个脚本会尝试读取你提供的绝对路径。如果文件不存在，或者脚本没有足够的权限读取它们，它会打印相应的错误信息。\n" +
                "2.  **编码**：默认使用 `utf-8` 编码读取文件，这是处理大多数文本文件的最佳实践。\n" +
                "\n" +
                "```python\n" +
                "import os\n" +
                "\n" +
                "def read_and_print_file(file_path):\n" +
                "    \"\"\"\n" +
                "    读取指定路径的文件内容并打印。\n" +
                "    如果文件不存在或无法读取，则打印错误信息。\n" +
                "    \"\"\"\n" +
                "    print(f\"\\n--- 正在读取文件: {file_path} ---\")\n" +
                "    if not os.path.exists(file_path):\n" +
                "        print(f\"错误: 文件不存在于此路径: {file_path}\")\n" +
                "        print(f\"--- 文件读取结束: {file_path} (未找到) ---\\n\")\n" +
                "        return\n" +
                "\n" +
                "    try:\n" +
                "        with open(file_path, 'r', encoding='utf-8') as f:\n" +
                "            content = f.read()\n" +
                "            print(content)\n" +
                "    except PermissionError:\n" +
                "        print(f\"错误: 没有权限读取文件: {file_path}\")\n" +
                "    except Exception as e:\n" +
                "        print(f\"读取文件时发生未知错误 {file_path}: {e}\")\n" +
                "    finally:\n" +
                "        print(f\"--- 文件读取结束: {file_path} ---\\n\")\n" +
                "\n" +
                "# 定义要读取的两个文件路径\n" +
                "file_path_1 = \"/Users/shenjiawei/run/py/open_ai.py\"\n" +
                "file_path_2 = \"/Users/shenjiawei/run/ws/ws.ini\"\n" +
                "\n" +
                "# 调用函数读取并打印第一个文件\n" +
                "read_and_print_file(file_path_1)\n" +
                "\n" +
                "# 调用函数读取并打印第二个文件\n" +
                "read_and_print_file(file_path_2)\n" +
                "```\n" +
                "\n" +
                "**如何运行这个脚本：**\n" +
                "\n" +
                "1.  将上述代码保存为一个 `.py` 文件，例如 `read_files.py`。\n" +
                "2.  打开你的终端或命令行。\n" +
                "3.  导航到你保存 `read_files.py` 文件的目录。\n" +
                "4.  运行命令：`python read_files.py`\n" +
                "\n" +
                "脚本将依次尝试读取这两个文件，并将其内容（或错误信息）打印到你的终端上。";
        Assert.assertEquals(expect, stream.getContentBuffer().toString());
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testStreamWithStreamFunCall() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        List<ProviderFunCallRequest> providerFunRequestList = new ArrayList<>();
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void addFunRequest(ProviderFunCallRequest providerFunRequest) {
                providerFunRequestList.add(providerFunRequest);
            }

            @Override
            protected void responseCheck() throws Exception {
                if (CollectionUtils.isEmpty(providerFunRequestList)) {
                    throw new WorkflowException("The response could not be parsed because it contains no content", ProtocolCode.C914);
                }
            }


            @Override
            protected void storeConversation(String content) {

            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        String content = IOUtils.toString(new BufferedReader(new InputStreamReader(ResourceUtils.getURL("classpath:VertexFunCalls.json").openStream(), "UTF-8")));
        stream.atonce(content.substring(1, content.length() - 1));
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(providerFunRequestList.size()));
        ProviderFunCallRequest p1 = providerFunRequestList.get(0);
        ProviderFunCallRequest p2 = providerFunRequestList.get(1);
        Assert.assertNotNull(p1.getCreated());
        Assert.assertEquals(p1.getName(), "workflow2");
        Assert.assertEquals(JsonUtils.write(p1.getArgs()), "{\"location\":\"华北\"}");
        Assert.assertEquals(JsonUtils.write(p1.getRefer()), "{\"functionCall\":{\"name\":\"workflow2\",\"args\":{\"location\":\"华北\"}}}");
        Assert.assertNotNull(p2.getCreated());
        Assert.assertEquals(p2.getName(), "workflow4");
        Assert.assertEquals(JsonUtils.write(p2.getArgs()), "{\"location\":\"华北\",\"index\":100}");
        Assert.assertEquals(JsonUtils.write(p2.getRefer()), "{\"functionCall\":{\"name\":\"workflow4\",\"args\":{\"location\":\"华北\",\"index\":100}}}");
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testVertexFunCallOnce() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall.json"));
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        StringBuilder buffer = new StringBuilder();
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(new NotifierServiceImpl() {
                    @Override
                    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                    }
                })
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build()) {

            @Override
            protected void addFunRequest(ProviderFunCallRequest providerFunRequest) throws Exception {
                buffer.append(providerFunRequest.getName() + providerFunRequest.getArgs()).append("\n");
            }

            @Override
            protected void responseCheck() throws Exception {
            }

            @Override
            protected void storeConversation(String content) {
            }
        };
        stream.atonce(json);
        Assert.assertEquals("_TDCASyhjQBATQjTTS{start=0, q=前端工程师 招聘 JD LinkedIn, num=10}\n" +
                "_TDCASyhjQBATQjTTS{start=0, q=前端工程师 招聘 JD 拉勾, num=10}\n" +
                "_TDCASyhjQBATQjTTS{num=10, q=前端工程师 招聘 JD BOSS直聘, start=0}\n" +
                "_TDCASyhjQBATQjTTS{q=前端工程师 招聘 JD 招聘网站, num=10, start=0}\n" +
                "_TDCASyhjQBATQjTTS{num=10, start=0, q=前端工程师 招聘 JD}\n" +
                "_TDCASyhjQBATQjTTS{start=0, q=前端开发工程师 招聘 JD LinkedIn, num=10}\n" +
                "_TDCASyhjQBATQjTTS{start=0, q=前端开发工程师 招聘 JD 拉勾, num=10}\n" +
                "_TDCASyhjQBATQjTTS{start=0, q=前端开发工程师 招聘 JD BOSS直聘, num=10}\n" +
                "_TDCASyhjQBATQjTTS{start=0, q=前端开发工程师 招聘 JD 招聘网站, num=10}\n" +
                "_TDCASyhjQBATQjTTS{q=前端开发工程师 招聘 JD, start=0, num=10}".trim(), buffer.toString().trim());
        EasyMock.verify(t);
    }

    /**
     * GoogleStream#atonce 在构建 {@link ProviderFunCallRequest} 时写入 {@code .model(this.r.getModel())}。
     */
    @Test
    public void testAtonce_funCallRequestCarriesProviderModel() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall.json"));
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        googleRequest.setModel("gemini-pro-from-request");
        List<String> models = new ArrayList<>();
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(new NotifierServiceImpl() {
                    @Override
                    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                    }
                })
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build()) {

            @Override
            protected void addFunRequest(ProviderFunCallRequest providerFunRequest) {
                models.add(providerFunRequest.getModel());
            }

            @Override
            protected void responseCheck() throws Exception {
                if (CollectionUtils.isEmpty(models)) {
                    throw new WorkflowException("The response could not be parsed because it contains no content", ProtocolCode.C914);
                }
            }

            @Override
            protected void storeConversation(String content) {
            }
        };
        stream.atonce(json);
        Assert.assertFalse(models.isEmpty());
        for (String m : models) {
            Assert.assertEquals("gemini-pro-from-request", m);
        }
        EasyMock.verify(t);
    }

    @Test
    public void testVertexFunCallOnce2() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall_2.json"));
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        StringBuilder buffer = new StringBuilder();
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(new NotifierServiceImpl() {
                    @Override
                    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                    }
                })
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build()) {

            @Override
            protected void addFunRequest(ProviderFunCallRequest providerFunRequest) {
                buffer.append(providerFunRequest.getName() + providerFunRequest.getArgs()).append("\n");
            }

            @Override
            protected void responseCheck() throws Exception {
                if (buffer.isEmpty()) {
                    throw new WorkflowException("The response could not be parsed because it contains no content", ProtocolCode.C914);
                }
            }

            @Override
            protected void storeConversation(String content) {
            }
        };
        stream.atonce(json);
        Assert.assertEquals("Tools_yCiDABCTxQhygwwBAgwTTRhgQzxhADSi{q=最新AI行业趋势 2023-2024}".trim(), buffer.toString().trim());
        EasyMock.verify(t);
    }

    @Test
    public void testOnceWithFinishMessage() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build());
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:VertexError.json"), StandardCharsets.UTF_8);
        try {
            stream.atonce(request);
            Assert.fail();
        } catch (WorkflowException e) {
            EasyMock.verify(s, t, c);
            Assert.assertEquals("Malformed function call: ` 工具只能执行独立的 Python 脚本，而不能启动一个持续运行的 Web 服务。因此，我将为您提供完整的 FastAPI 代码，并解释如何运行它。在 `", e.getMessage());
        }
    }

    @Test
    public void testStreamWithFinishMessage() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }
        };
        String request = IOUtils.toString(ResourceUtils.getURL("classpath:VertexError_stream.json"), StandardCharsets.UTF_8);
        try {
            List<Object> data = JsonUtils.read(request, List.class);
            boolean finish = false;
            for (Object each : data) {
                finish = finish || stream.stream(JsonUtils.write(each));
            }
            Assert.fail();
        } catch (WorkflowException e) {
            EasyMock.verify(s, t, c);
            Assert.assertEquals("Malformed function call: ` 工具只能执行独立的 Python 脚本，而不能启动一个持续运行的 Web 服务。因此，我将为您提供完整的 FastAPI 代码，并解释如何运行它。在 `", e.getMessage());
        }
    }

    @Test
    public void testCallbackAtOnceWithMaxToken() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        try {
            stream.callback(IOUtils.toString(ResourceUtils.getURL("classpath:VertexError_maxtoken.json").openStream(), StandardCharsets.UTF_8));
        } catch (Exception e) {
            Assert.assertEquals("Vertex encountered an exception: MAX_TOKENS", e.getMessage());
        } finally {
            EasyMock.verify(signal, store, trackService, request);
        }
    }

    @Test
    public void testCallbackStreamWithMaxToken() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        request.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        try {
            stream.callback(IOUtils.toString(ResourceUtils.getURL("classpath:VertexError_maxtoken.json").openStream(), StandardCharsets.UTF_8));
        } catch (Exception e) {
            Assert.assertEquals("Vertex encountered an exception: MAX_TOKENS", e.getMessage());
        } finally {
            EasyMock.verify(signal, store, trackService, request);
        }
    }

    @Test
    public void testVertexFunCallThoughtSignature1() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall_thoughtSignature.json"));
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithNothing())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build()) {

            @Override
            protected String getFunData(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
                return "";
            }

            @Override
            protected void storeConversation(String content) {
            }
        };
        stream.atonce(json);
        String response = "{\"functionCall\":{\"name\":\"Tools_shell__cmd\",\"args\":{\"cmd\":\"cd ~/DEV/gocli && go mod init gocli || true\",\"why_do_this\":\"初始化Go模块，确保依赖管理正常。如果已经初始化过，忽略错误。\"}},\"thoughtSignature\":\"CicB4/H/Xu2CMtiGwct5OIZzEfBky2t72cAwY9LURpl2tPU7sqZ64CAKXQHj8f9eVrAZOIz4GNN6vg6e9SdOTbJ7gdZcJkjyjiZtdzoHVtKCc8tmjdQmgQ9+QWNrBS9cYmt2BjZ5vUF3wr5yxA9luVkFWUcggGabd4azrLsKyfti5YBBsRGfrApRAePx/14Rp8z+KCyh9uLfPMEkTQCIVomSO0r7xnfNfm2TbiHiEFtWr1cWrABmOrSSMY81zXFl6IT3s8RUwucxQuC1Q8lZ0dLjFkBxg3G06lOxCnQB4/H/Xq8iZpzhDvl7LF7gc25jVcmUqWSdo0TXTwD89Uzt9WyZiIwEcToECX8cTq3lxqi109CkCgtqaMDdoH+K1n5Sv7SSr6JfsTgAMBeIFNI3TjW6pMQY4wdZ07LHkmoV+W+g67ozICZJ8oGX5oiEYnJ56QqJAQHj8f9efd6C025v5fjQ0RpXOZTh+nfZ/tJJXRRFg+sKhCr57mumDtQGipQTnWZiR+IkwoptFEVgsvldF+nLuE/i+Rcw8kQdLurjYWqjt/f4yi7Z+aCoH3U/RDE3bppjIZynzUdOISxSRp+QDWEP4Kwjk1bXGYFdtSpYm1rpEjiVl1JwK0OtLl6hCs0BAePx/141izGDAiBQOsmltexhdQytFFO6xp0acOvEnM32oPohikfL3/OQ95EE0YDY2BReZwBW0pNAxIjGAwVRm9J4upC2z811SBAgYsBc7C0dlXqVA8Wh5je9Hop4x/GwnC/81sOaaRME+RKynM62V7rHhtjCvXy7/67m/O9SNxKnjUSR0D3dd0gPvBc6gbrlB3JG08M0RcLfUPsq3OJsYr/hOuQmWYWOa4ckhafoivEGG4COMli4Mx9q9InnDX5eyPdk1dygpSLZbSCcIQr7AQHj8f9eZtfxsF0zrtkBgmz+2TImSYUdRyp5+AUF4NyBubtehMU48WzDwsdhar3sLQ7UAMSFEgHAgNslc8o6TXIq/aFyyuOWAelKO4eb2HMaaRRXfpInvAdIxsMWYyq3uEW4oAjXgOHHNnYFPdo9kn3xOUNa9b3u45uo7o0431ik5BWfv5wWy3DVc5JoJ1cDS+XJLfRz1hDkpVKppz64d7V26tLjdqIH4DWbxP5yctpYTgqdOd3mS9/xIQ0aPDPpijqgVQVVoxnYz4+1h3WyswfonkI2xWyXuRjjfqMkHjMQeakfq9mNJ/w1AaWPRyvBku51bDLQj87ooRlN\"}";
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(stream.getProviderFunRequests().size()));
        Assert.assertEquals(response, JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()));
        EasyMock.verify(t);
    }

    @Test
    public void testVertexFunCallThoughtSignature2() throws Exception {
        String json = IOUtils.toString(ResourceUtils.getURL("classpath:VertexFunCall_multi.json"));
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithNothing())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build()) {

            @Override
            protected void addFunRequest(ProviderFunCallRequest providerFunRequest) throws Exception {
                Assert.assertNotNull(MapUtils.getString(providerFunRequest.getRefer(), "thoughtSignature"));
            }

            @Override
            protected void responseCheck() throws Exception {
            }

            @Override
            protected void storeConversation(String content) {
            }
        };
        List<Object> data = JsonUtils.read(json, List.class);
        for (Object each : data) {
            stream.stream(JsonUtils.write(each));
        }
        EasyMock.verify(t);
    }

    @Test
    public void testOnceWithImage1() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(false).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl() {
            @Override
            public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
                String file = super.write(mediaInlineData, workTask);
                new File(file).delete();
                return file;
            }
        };
        mediaInlineService.setFileStore(ObjectBuilder.buildFileStore());
        mediaInlineService.init();
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(mediaInlineService)
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void addInlineData(Map<String, Object> data) throws Exception {
                Assert.assertEquals(data.get("mimeType"), "image/png");
                Assert.assertNotNull(data.get("data"));
                super.addInlineData(data);
            }

            @Override
            protected String addContent(String text, Boolean finished) throws Exception {
                Assert.assertNotNull(text);
                return this.contentBuffer.append(text).toString();
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("I'm now carefully checking the generated image against the original prompt. The image of Pikachu in a forest is a good fit. I'm focusing on the alignment between the image's elements and the user's description. The image seems to capture the essence of what was requested"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.atonce(IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse_image.json").openStream(), StandardCharsets.UTF_8)));
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testOnceWithImage2() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        h.store(EasyMock.anyObject(Dimension.class), EasyMock.anyObject(List.class), EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl() {
            @Override
            public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
                String file = super.write(mediaInlineData, workTask);
                new File(file).deleteOnExit();
                return file;
            }
        };
        mediaInlineService.setFileStore(ObjectBuilder.buildFileStore());
        mediaInlineService.init();
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(mediaInlineService)
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void addInlineData(Map<String, Object> data) throws Exception {
                Assert.assertEquals(data.get("mimeType"), "image/png");
                Assert.assertNotNull(data.get("data"));
                super.addInlineData(data);
            }

            @Override
            protected String addContent(String text, Boolean finished) throws Exception {
                Assert.assertNotNull(text);
                return this.contentBuffer.append(text).toString();
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("I'm now carefully checking the generated image against the original prompt. The image of Pikachu in a forest is a good fit. I'm focusing on the alignment between the image's elements and the user's description. The image seems to capture the essence of what was requested"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.atonce(IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse_image.json").openStream(), StandardCharsets.UTF_8)));
        EasyMock.verify(s, t, c);
    }

    @Test(expected = WorkflowException.class)
    public void testStreamEmptyCandidates() throws Exception {
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build());
        stream.stream("{\"candidates\":[]}");
    }

    @Test
    public void testTokenStatisticZero() throws Exception {
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build());
        stream.tokenStatistic(Collections.singletonMap("usageMetadata", Collections.singletonMap("totalTokenCount", 0)));
    }

    /**
     * 覆盖 GoogleStream#tokenStatistic：body 含 usageMetadata 且 totalTokenCount != 0 时调用 tokenStatistic.stat 并设置 segment.usage
     */
    @Test
    public void testTokenStatistic_withTotalTokenCount_callsStatAndSetsSegmentUsage() throws Exception {
        GoogleRequest googleRequest = new GoogleRequest();
        googleRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TokenStatistic tokenStatistic = ObjectBuilder.buildTokenStatistic();
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(tokenStatistic)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(googleRequest)
                .build());
        Map<String, Object> usageMetadata = new HashMap<>();
        usageMetadata.put("promptTokenCount", 50);
        usageMetadata.put("cachedContentTokenCount", 10);
        usageMetadata.put("thoughtsTokenCount", 5);
        usageMetadata.put("totalTokenCount", 100);
        Map<String, Object> body = new HashMap<>();
        body.put("usageMetadata", usageMetadata);
        stream.tokenStatistic(body);
        Assert.assertNotNull(stream.getSegment().getUsage());
        Assert.assertEquals(Integer.valueOf(100), stream.getSegment().getUsage().getTotal());
    }

    @Test
    public void testStoreCompleted() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        h.store(EasyMock.anyObject(Dimension.class), EasyMock.anyObject(List.class), EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(false).anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl() {
            @Override
            public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
                String file = super.write(mediaInlineData, workTask);
                new File(file).deleteOnExit();
                return file;
            }
        };
        mediaInlineService.setFileStore(ObjectBuilder.buildFileStore());
        mediaInlineService.init();
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(mediaInlineService)
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void addInlineData(Map<String, Object> data) throws Exception {
                Assert.assertEquals(data.get("mimeType"), "image/png");
                Assert.assertNotNull(data.get("data"));
                super.addInlineData(data);
            }

            @Override
            protected String addContent(String text, Boolean finished) throws Exception {
                Assert.assertNotNull(text);
                return this.contentBuffer.append(text).toString();
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(historyPairs.size()));
                Assert.assertTrue(historyPairs.getFirst().getAnswer().contains("I'm now carefully checking the generated image against the original prompt. The image of Pikachu in a forest is a good fit. I'm focusing on the alignment between the image's elements and the user's description. The image seems to capture the essence of what was requested"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getFirst().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.atonce(IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse_image.json").openStream(), StandardCharsets.UTF_8)));
        EasyMock.verify(s, t, c);
    }

    @Test
    public void testStreamWithImage1() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        GoogleRequest c = EasyMock.createMock(GoogleRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(false).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl() {
            @Override
            public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
                String file = super.write(mediaInlineData, workTask);
                new File(file).delete();
                return file;
            }
        };
        mediaInlineService.setFileStore(ObjectBuilder.buildFileStore());
        mediaInlineService.init();
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(mediaInlineService)
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void notify(int seqid, boolean finished) throws Exception {
                Assert.assertFalse(finished);
                super.notify(seqid, finished);
            }

            @Override
            protected void addInlineData(Map<String, Object> data) throws Exception {
                Assert.assertEquals(data.get("mimeType"), "image/png");
                Assert.assertNotNull(data.get("data"));
                super.addInlineData(data);
            }

            @Override
            protected String addContent(String text, Boolean finished) throws Exception {
                Assert.assertNotNull(text);
                return this.contentBuffer.append(text).toString();
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("I'm now carefully checking the generated image against the original prompt. The image of Pikachu in a forest is a good fit. I'm focusing on the alignment between the image's elements and the user's description. The image seems to capture the essence of what was requested"));
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        Assert.assertTrue(stream.stream(IOUtils.toString(ResourceUtils.getURL("classpath:VertexResponse_image.json").openStream(), StandardCharsets.UTF_8)));
        EasyMock.verify(s, t, c);
    }
}
