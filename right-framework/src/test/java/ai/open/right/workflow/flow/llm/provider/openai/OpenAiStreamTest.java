package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQueryDelegate;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderData;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.*;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class OpenAiStreamTest {

    @Test
    public void testTools() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
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
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifyProcess() throws Exception {
                Assert.assertEquals(1, this.providerFunRequests.size());
            }
        };
        stream.stream(IOUtils.toString(ResourceUtils.getURL("classpath:OpenAiFunCalls.json").openStream()));
        EasyMock.verify(s, h, c, t);
    }

    @Test
    public void testStatus() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
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
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(1042), tokenData.getTotal());
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
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(c.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertEquals("TOKEN", historyPairs.getLast().getAnswer());
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        stream.stream("data: " + IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_token.json").openStream()));
        EasyMock.verify(s, h, c, t);
    }

    @Test
    public void testTokenStatistic() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall().anyTimes();
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
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
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(5100), tokenData.getTotal());
                Assert.assertEquals(Integer.valueOf(3840), tokenData.getCache());
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
                .build());
        Map<String, Object> usage = JsonUtils.read("{\"prompt_tokens\":5000,\"completion_tokens\":100,\"total_tokens\":5100,\"prompt_tokens_details\":{\"cached_tokens\":3840},\"completion_tokens_details\":{\"reasoning_tokens\":0,\"accepted_prediction_tokens\":0,\"rejected_prediction_tokens\":0}}\n", Map.class);
        stream.tokenStatistic(ImmutableMap.of("usage", usage));
        Assert.assertEquals(Integer.valueOf(3840), stream.getSegment().getUsage().getCache());
        Assert.assertEquals(Integer.valueOf(5100), stream.getSegment().getUsage().getTotal());
        EasyMock.verify(s, h, c, t);
    }

    /**
     * 覆盖 OpenAiStream#tokenStatistic：body 含 usage（prompt_tokens_details.cached_tokens、completion_tokens_details.reasoning_tokens、total_tokens、prompt_tokens），
     * 当 cache!=0 或 total!=0 时调用 tokenStatistic.stat 并设置 segment.usage（cache、total）
     */
    @Test
    public void testTokenStatistic_withUsageStructure_callsStatAndSetsSegmentUsage() throws Exception {
        OpenAiRequest c = new OpenAiRequest();
        c.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("prompt_tokens", 70);
        usage.put("total_tokens", 100);
        Map<String, Object> promptDetails = new HashMap<>();
        promptDetails.put("cached_tokens", 30);
        usage.put("prompt_tokens_details", promptDetails);
        Map<String, Object> completionDetails = new HashMap<>();
        completionDetails.put("reasoning_tokens", 5);
        usage.put("completion_tokens_details", completionDetails);
        Map<String, Object> body = new HashMap<>();
        body.put("usage", usage);
        stream.tokenStatistic(body);
        Assert.assertNotNull(stream.getSegment().getUsage());
        Assert.assertEquals(Integer.valueOf(30), stream.getSegment().getUsage().getCache());
        Assert.assertEquals(Integer.valueOf(100), stream.getSegment().getUsage().getTotal());
    }

    /**
     * 覆盖 OpenAiStream#tokenStatistic：cache==0 且 total==0 时不调用 stat、不设置 segment.usage，不抛异常
     */
    @Test
    public void testTokenStatistic_zeroCacheAndTotal_doesNotStat() throws Exception {
        OpenAiRequest c = new OpenAiRequest();
        c.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("prompt_tokens", 0);
        usage.put("total_tokens", 0);
        usage.put("prompt_tokens_details", Collections.emptyMap());
        usage.put("completion_tokens_details", Collections.emptyMap());
        Map<String, Object> body = new HashMap<>();
        body.put("usage", usage);
        stream.tokenStatistic(body);
        Assert.assertNull(stream.getSegment().getUsage());
    }

    @Test
    public void testOnceFunCall() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
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
        EasyMock.expect(c.getFunCallTimeout()).andReturn(10086).anyTimes();
        EasyMock.expect(c.isTakeover("Tools_RxSTQRzCBwxARDxjyBihigBxxzBSjBCj")).andReturn(false).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(new LLMQueryDelegate(ObjectBuilder.buildWorkflowTaskWithTimestamp(10086L), "", ""))).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifyProcess() throws Exception {

            }
        };
        String response = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_funcall.json").openStream(), StandardCharsets.UTF_8);
        Assert.assertTrue(stream.atonce(response));
        Assert.assertEquals(Integer.valueOf(338), Integer.valueOf(JsonUtils.write(stream.getProviderFunRequests()).length()));
        EasyMock.verify(s, h, c, t);
    }

    @Test
    public void testStreamFunCall() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
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
        EasyMock.expect(c.getFunCallTimeout()).andReturn(10086).anyTimes();
        EasyMock.expect(c.isTakeover("Tools_RxSTQRzCBwxARDxjyBihigBxxzBSjBCj")).andReturn(false).anyTimes();
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
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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

            }

            @Override
            protected void notifyProcess() throws Exception {

            }
        };
        String response1 = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_funcall_s1.json").openStream(), StandardCharsets.UTF_8);
        Assert.assertFalse(stream.stream("data: " + response1));
        String response2 = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_funcall_s2.json").openStream(), StandardCharsets.UTF_8);
        Assert.assertFalse(stream.stream("data: " + response2));
        String response3 = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_funcall_s3.json").openStream(), StandardCharsets.UTF_8);
        Assert.assertTrue(stream.stream("data: " + response3));
        Assert.assertEquals(Integer.valueOf(334), Integer.valueOf(JsonUtils.write(stream.getProviderFunRequests()).length()));
        EasyMock.verify(s, h, c, t);
    }

    @Test
    public void testStoreFunCallAtOnce() throws Exception {
        NamesServiceImpl namesService = NamesServiceImpl.class.cast(ObjectBuilder.buildNamesService());
        namesService.setEncode(false);
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getFunCallHeritage()).andReturn(false).anyTimes();
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(10086).anyTimes();
        EasyMock.expect(c.isTakeover("Tools___RxSTQRzCBwxARDxjyBihigBxxzBSjBCj")).andReturn(false).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        h.store(EasyMock.anyObject(), EasyMock.anyObject(List.class), EasyMock.anyObject(HistoryPair.class), EasyMock.anyInt(), EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(namesService)
                .request(c)
                .build()) {
            protected void notifySegment() throws Exception {

            }
        };
        String response = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_funcall.json").openStream(), StandardCharsets.UTF_8);
        Assert.assertTrue(stream.atonce(response));
        EasyMock.verify(s, h, c, t);
    }

    @Test
    public void testStoreFunCallStream() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getApi()).andReturn("API").anyTimes();
        EasyMock.expect(c.getModel()).andReturn("MODEL").anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(10086).anyTimes();
        EasyMock.expect(c.isTakeover("Tools_RxSTQRzCBwxARDxjyBihigBxxzBSjBCj")).andReturn(false).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        h.store(EasyMock.anyObject(), EasyMock.anyObject(List.class), EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {

            }
        };
        String response = IOUtils.toString(ResourceUtils.getURL("classpath:OpenAIResponse_funcall_s2.json").openStream(), StandardCharsets.UTF_8);
        Assert.assertFalse(stream.stream("data: " + response));
        EasyMock.verify(s, h, c, t);
    }


    @Test
    public void testCallbackWithNormalMessage() throws Exception {
        String message = "{\"text\": \"Hello\"}";
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        c.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.getProviderData()).andReturn(new ProviderData()).anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(10086).anyTimes();
        EasyMock.expect(c.isTakeover("Tools_RxSTQRzCBwxARDxjyBihigBxxzBSjBCj")).andReturn(false).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        h.store(EasyMock.anyObject(), EasyMock.anyObject(List.class), EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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
            public Boolean stream(String source) throws Exception {
                Assert.assertEquals(message, source);
                return false;
            }
        };
        stream.callback(message);
    }

    @Test
    public void testCallbackWithDone() throws Exception {
        String message = "{\"text\": \"Hello\"}";
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(10086).anyTimes();
        EasyMock.expect(c.isTakeover("Tools_RxSTQRzCBwxARDxjyBihigBxxzBSjBCj")).andReturn(false).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        h.store(EasyMock.anyObject(), EasyMock.anyObject(List.class), EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyInt());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.fail();
            }
        };
        stream.callback(ProviderReader.DONE);
    }
}
