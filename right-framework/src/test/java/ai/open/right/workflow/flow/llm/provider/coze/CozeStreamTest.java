package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class CozeStreamTest {

    @Test
    public void testOnceWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        CozeRequest coze = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(coze.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(coze.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(coze.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(coze.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(coze.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(coze.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(coze.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(coze.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(coze.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(coze.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(coze.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(coze.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(coze.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(coze.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(coze.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(s, coze);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(coze.getMessage(), coze.getRepositories(), mockHistories, coze.getExpired(), coze.getHistories());
        EasyMock.expectLastCall().anyTimes();
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t, h);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(coze)
                .build()) {
            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("UNKNOWN", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(coze.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertEquals("@@@@@@@@", historyPairs.getLast().getAnswer());
                Assert.assertTrue(Long.valueOf(coze.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        boolean rst = stream.atonce("{\"messages\":[{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"@@@@@@@@\",\"content_type\":\"text\"},{\"role\":\"assistant\",\"type\":\"verbose\",\"content\":\"{\\\"msg_type\\\":\\\"generate_answer_finish\\\",\\\"data\\\":\\\"{\\\\\\\"finish_reason\\\\\\\":0}\\\",\\\"from_module\\\":null,\\\"from_unit\\\":null}\",\"content_type\":\"text\"}],\"conversation_id\":\"123\",\"code\":0,\"msg\":\"success\"}");
        Assert.assertTrue(rst);
        EasyMock.verify(s, trackService, coze);
        stream.tokenStatistic(null);
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithJsonError() throws Exception {
        NotifierServiceImpl r = EasyMock.createMock(NotifierServiceImpl.class);
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(r, s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
        try {
            stream.atonce("\"messages\":[{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"@@@@@@@@\",\"content_type\":\"text\"},{\"role\":\"assistant\",\"type\":\"verbose\",\"content\":\"{\\\"msg_type\\\":\\\"generate_answer_finish\\\",\\\"data\\\":\\\"{\\\\\\\"finish_reason\\\\\\\":0}\\\",\\\"from_module\\\":null,\\\"from_unit\\\":null}\",\"content_type\":\"text\"}],\"conversation_id\":\"123\",\"code\":0,\"msg\":\"success\"}");
        } finally {
            EasyMock.verify(r, s, t, h, c);
        }
    }

    @Test(expected = MismatchedInputException.class)
    public void testOnceWithFinishedWithNull() throws Exception {
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        try {
            CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
                    .trackFunCallService(t)
                    .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                    .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                    .notifierService(null)
                    .providerReason(ObjectBuilder.getProviderReason())
                    .signalStream(s)
                    .historyStore(h)
                    .namesService(ObjectBuilder.buildNamesService())
                    .request(c)
                    .build());
            stream.atonce("\"messages\":[{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"@@@@@@@@\",\"content_type\":\"text\"},{\"role\":\"assistant\",\"type\":\"verbose\",\"content\":\"{\\\"msg_type\\\":\\\"generate_answer_finish\\\",\\\"data\\\":\\\"{\\\\\\\"finish_reason\\\\\\\":0}\\\",\\\"from_module\\\":null,\\\"from_unit\\\":null}\",\"content_type\":\"text\"}],\"conversation_id\":\"123\",\"code\":0,\"msg\":\"success\"}");
        } finally {
            EasyMock.verify(s, t, h, c);
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testOnceWithFinishedWithEmptyMessage() throws Exception {
        NotifierServiceImpl r = EasyMock.createMock(NotifierServiceImpl.class);
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(r, s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
        try {
            stream.atonce("{\"message_s\":[{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"@@@@@@@@\",\"content_type\":\"text\"},{\"role\":\"assistant\",\"type\":\"verbose\",\"content\":\"{\\\"msg_type\\\":\\\"generate_answer_finish\\\",\\\"data\\\":\\\"{\\\\\\\"finish_reason\\\\\\\":0}\\\",\\\"from_module\\\":null,\\\"from_unit\\\":null}\",\"content_type\":\"text\"}],\"conversation_id\":\"123\",\"code\":0,\"msg\":\"success\"}");
        } finally {
            EasyMock.verify(r, s, t, h, c);
        }
    }

    @Test
    public void testStreamWithNotFinished() throws Exception {
        NotifierServiceImpl r = EasyMock.createMock(NotifierServiceImpl.class);
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(10).anyTimes();
        EasyMock.replay(r, s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
        boolean rst = stream.stream("data: {\"event\":\"message\",\"message\":{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"里\",\"content_type\":\"text\"},\"is_finish\":false,\"index\":0,\"conversation_id\":\"123\",\"seq_id\":206}");
        Assert.assertFalse(rst);
        EasyMock.verify(r, s, t, h, c);
    }

    @Test
    public void testStreamWithFinishedAndDone() throws Exception {
        NotifierServiceImpl r = EasyMock.createMock(NotifierServiceImpl.class);
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(10).anyTimes();
        EasyMock.replay(r, s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
        boolean rst = stream.stream("data: {\"event\":\"done\",\"message\":{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"里\",\"content_type\":\"text\"},\"is_finish\":false,\"index\":0,\"conversation_id\":\"123\",\"seq_id\":206}");
        Assert.assertTrue(rst);
        EasyMock.verify(r, s, t, h, c);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testStreamWithInVaildBody() throws Exception {
        NotifierServiceImpl r = EasyMock.createMock(NotifierServiceImpl.class);
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(10).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(r, s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
        try {
            stream.stream("{\"event\":\"message\",\"message\":{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"里\",\"content_type\":\"text\"},\"is_finish\":false,\"index\":0,\"conversation_id\":\"123\",\"seq_id\":206}");
        } finally {
            EasyMock.verify(r, s, h, t, c);
        }
    }

    @Test(expected = MismatchedInputException.class)
    public void testStreamWithInVaildJson() throws Exception {
        NotifierServiceImpl r = EasyMock.createMock(NotifierServiceImpl.class);
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithNothing();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(10).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(r, s, h, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
        try {
            stream.stream("data: \"event\":\"done\",\"message\":{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"里\",\"content_type\":\"text\"},\"is_finish\":false,\"index\":0,\"conversation_id\":\"123\",\"seq_id\":206}");
        } finally {
            EasyMock.verify(r, s, h, t, c);
        }
    }


    @Test
    public void testStreamWithFinished() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        CozeRequest c = EasyMock.createMock(CozeRequest.class);
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(10).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.replay(c, s);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(c.getMessage(), c.getRepositories(), mockHistories, c.getExpired(), c.getHistories());
        EasyMock.expectLastCall().anyTimes();
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t, h);
        CozeStream stream = new CozeStream(ProviderStreamConfig.<CozeRequest>builder()
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
                Assert.assertEquals("UNKNOWN", historyPairs.getLast().getAnswer());
                Assert.assertTrue(Long.valueOf(c.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        boolean rst = stream.stream("data: {\"event\":\"message\",\"message\":{\"role\":\"assistant\",\"type\":\"answer\",\"content\":\"UNKNOWN\",\"content_type\":\"text\"},\"is_finish\":true,\"index\":0,\"conversation_id\":\"123\",\"seq_id\":206}");
        Assert.assertTrue(rst);
        EasyMock.verify(s, t, c);
    }
}
