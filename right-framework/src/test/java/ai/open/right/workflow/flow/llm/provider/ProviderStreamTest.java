package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.TakeoverException;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMTakeover;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCall;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.lang.reflect.Field;
import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;

public class ProviderStreamTest {

    @Test
    public void testInit1() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        // Nothing
        stream.tokenStatistic(null);
        Assert.assertNotNull(stream.getMediaInlineService());
        Assert.assertNotNull(stream.getProviderReason());
        Assert.assertNotNull(stream.getTokenStatistic());
        Assert.assertNotNull(stream.getTrackFunCallService());
        Assert.assertNotNull(stream.getNotifierService());
        Assert.assertNotNull(stream.getHistoryStore());
        Assert.assertNotNull(stream.getSignalStream());
        Assert.assertNotNull(stream.getSegment());
        Assert.assertNotNull(stream.getRequest());
        EasyMock.verify(signal, store, trackService, request);
        Assert.assertNull(stream.getProviderFunRequests());
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        stream.setProviderFunRequests(providerFunRequests);
        Assert.assertNotNull(stream.getProviderFunRequests());
    }

    @Test
    public void addReason_defaultImplementation_noOp() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        GoogleRequest request = new GoogleRequest();
        request.setPrefix("");
        request.setSuffix("");
        request.setStream(true);
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        EasyMock.replay(signal, store);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        Assert.assertNull(stream.getReasoning());
        int contentLen = stream.getContentBuffer().length();
        stream.addReason(ImmutableMap.of("reasoning_content", "any"), false);
        stream.addReason(new HashMap<>(), true);
        Assert.assertNull(stream.getReasoning());
        Assert.assertEquals(contentLen, stream.getContentBuffer().length());
        EasyMock.verify(signal, store, trackService);
    }

    @Test
    public void testInit2() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        Assert.assertEquals(Notifier.ENDPOINT, stream.getSegment().getNotifier());
    }

    @Test
    public void testInit3() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.putMetadata(ProviderRequestService.KEY_FUN_MERGE, true);
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        Assert.assertEquals(Notifier.ENDPOINT, stream.getSegment().getNotifier());
    }

    @Test
    public void testInit4() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getChain()).andReturn("CHAIN").anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        Assert.assertEquals(Notifier.LOCALHOST, stream.getSegment().getNotifier());
    }


    @Test
    public void testGetAndDelContentBuffer() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.contentBuffer.append("Hello World");
        Assert.assertEquals("llo ", stream.getAndDelContentBuffer(2, 6));
        Assert.assertEquals("HeWorld", stream.contentBuffer.toString());
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testIndexOfContentBuffer() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.contentBuffer.append("Hello World");
        Assert.assertEquals(Integer.valueOf(3), stream.indexOfContentBuffer("lo", 2));
        Assert.assertEquals(Integer.valueOf(3), stream.indexOfContentBuffer("lo"));
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testSetSignalMetadata() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.setSignalMetadata("OK_SIGNAL");
        Assert.assertEquals("OK_SIGNAL", List.class.cast(stream.segment.getMetadata().get(SignalExecutor.SIGNAL_KEY)).get(0));
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testSetOther() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.setWorkflow("WK");
        stream.setNotifier("NO");
        stream.silent(true);
        stream.notify(true);
        Assert.assertEquals("WK", stream.segment.getWorkflow());
        Assert.assertEquals("NO", stream.segment.getNotifier());
        Assert.assertTrue(stream.segment.getSilent());
        Assert.assertTrue(stream.notify);
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testCallbackStream() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {

            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.callback("HELLO");
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testCallbackStreamWithFalse() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {

            @Override
            protected Boolean stream(String source) {
                return false;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.callback("HELLO");
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testCallbackAtOnce() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.callback("HELLO");
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testCallbackAtOnceWithFalse() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        request.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(signal, store, request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return false;
            }
        };
        stream.callback("HELLO");
        EasyMock.verify(signal, store, trackService, request);
    }

    @Test
    public void testCallbackException() throws Exception {
        NotifierServiceImpl manager = ObjectBuilder.buildNotifierManagerWithimplement();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.finish(message);
        EasyMock.expectLastCall().anyTimes();
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
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
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(manager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signal)
                .historyStore(store)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {

            @Override
            protected Boolean stream(String source) throws Exception {
                throw new RuntimeException();
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                throw new RuntimeException("HELLO");
            }

        };
        try {
            providerStream.callback("WORLD");
            Assert.fail();
        } catch (Exception e) {
            Assert.assertTrue(e.getMessage().contains("HELLO"));
            EasyMock.verify(trackService, signal, store, request);
        }
    }

    @Test
    public void testWithNotifyTools() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        AtomicBoolean run = new AtomicBoolean();
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }

            protected void getFunResponse() throws Exception {
                run.set(true);
            }

            protected void notifySegment() throws Exception {

            }
        };
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().build();
        providerStream.addFunRequest(providerFunRequest);
        providerStream.notifyProcess();
        Assert.assertTrue(run.get());
        EasyMock.verify(request, trackService);
    }

    @Test
    public void testAddFunRequest() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().build();
        providerStream.addFunRequest(providerFunRequest);
        Assert.assertEquals(providerFunRequest, providerStream.providerFunRequests.getFirst());
        EasyMock.verify(request, trackService);
    }

    @Test
    public void testNotifyFunRequest() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getFunCallHeritage()).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME1")).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME2")).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME3")).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getMetadata()).andReturn(null).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        providerStream.addFunRequest(ProviderFunCallRequest.builder().args(Collections.singletonMap("HELLO1", "WORLD1")).name("NAME1").build());
        providerStream.addFunRequest(ProviderFunCallRequest.builder().args(Collections.singletonMap("HELLO2", "WORLD2")).name("NAME2").build());
        providerStream.addFunRequest(ProviderFunCallRequest.builder().name("NAME3").build());
        providerStream.getFunResponse();
        Assert.assertEquals("UNKNOWN", providerStream.contentBuffer.toString());
        EasyMock.verify(request, trackService);
    }

    @Test
    public void testGetFunRequest() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setWorkflow(ProviderRequestService.KEY_FUN_SELECT);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andReturn("HELLO").anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("WORLD").anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        String response = providerStream.getFunResponse(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("HELLOWORLD", response);
        EasyMock.verify(request, syncWorkflowTask1, syncWorkflowTask2, trackService);
    }

    @Test
    public void testGetFunRequestWithException1() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setWorkflow(ProviderRequestService.KEY_FUN_SELECT);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andReturn("HELLO").anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("WORLD").anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        String response = providerStream.getFunResponse(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("HELLOWORLD", response);
        EasyMock.verify(request, syncWorkflowTask1, syncWorkflowTask2, trackService);
    }

    @Test
    public void testGetFunRequestWithException2() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        EasyMock.expect(request.isTakeover("NAME1")).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME2")).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(10000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.expect(request.getMetadata()).andReturn(null).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLOWORLD"))
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        providerStream.providerFunRequests = new ArrayList();
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().name("NAME1").build();
        providerStream.providerFunRequests.add(providerFunRequest);
        providerFunRequest = ProviderFunCallRequest.builder().name("NAME2").build();
        providerStream.providerFunRequests.add(providerFunRequest);
        // just one
        providerStream.providerFunRequests.add(ProviderFunCallRequest.builder().name("NAME1").build());
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andReturn("HELLO").anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andThrow(new RuntimeException("ERROR")).anyTimes();
        EasyMock.expect(syncWorkflowTask1.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.expect(syncWorkflowTask2.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        String response = providerStream.getFunResponse(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("HELLOWORLD", response);
        EasyMock.verify(request, syncWorkflowTask1, syncWorkflowTask2, trackService);
    }

    @Test
    public void testTrack() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        trackService.store(EasyMock.anyObject(TrackFunCall.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.beginFunCallTrack("ABC");
        SyncConfig syncConfig = SyncConfig.builder().timeout(10000).workTask(workflowTask).build();
        SyncWorkflowTask syncWorkflowTask = SyncWorkflowTask.exeWorkflow(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), syncConfig);
        providerStream.trackFunCall(providerFunRequest, syncWorkflowTask);
        EasyMock.verify(request, trackService);
    }

    @Test
    public void testTrackWithException() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        trackService.store(EasyMock.anyObject(TrackFunCall.class));
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().build();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.beginFunCallTrack("ABC");
        SyncConfig syncConfig = SyncConfig.builder().timeout(10000).workTask(workflowTask).build();
        SyncWorkflowTask syncWorkflowTask = SyncWorkflowTask.exeWorkflow(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect(), syncConfig);
        providerStream.trackFunCall(providerFunRequest, syncWorkflowTask);
        EasyMock.verify(request, trackService);
    }

    @Test
    public void testGetFunResponseException() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("HELLO WORLD").anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        Assert.assertEquals("HELLO WORLD", providerStream.getFunSelect(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2)));
        EasyMock.verify(syncWorkflowTask1, syncWorkflowTask2);
    }

    /**
     * getFunSelect 中当某 task.get() 抛出 needSlient() 的 WorkflowException 时走静默分支，吞掉异常并继续，仅追加后续 task 结果
     */
    @Test
    public void testGetFunSelect_silentException_logsDebugNotError() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andThrow(new WorkflowException("task closed").needSilent()).anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("HELLO WORLD").anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        String result = providerStream.getFunSelect(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("HELLO WORLD", result);
        EasyMock.verify(syncWorkflowTask1, syncWorkflowTask2);
    }

    /**
     * trackFunCall 中 response.get() 抛出 needSlient() 的 WorkflowException 时走静默分支，不向外抛异常
     */
    @Test
    public void testTrackFunCall_silentException_logsDebugNotError() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask.containFunCallTrack()).andReturn(true).anyTimes();
        EasyMock.expect(syncWorkflowTask.getWorkTask()).andReturn(ObjectBuilder.buildWorkflowTask()).anyTimes();
        EasyMock.expect(syncWorkflowTask.getFunCallTrack()).andReturn("trackId").anyTimes();
        EasyMock.expect(syncWorkflowTask.get()).andThrow(new WorkflowException("task closed").needSilent()).anyTimes();
        EasyMock.replay(syncWorkflowTask);
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().name("FUN").build();
        providerStream.trackFunCall(providerFunRequest, syncWorkflowTask);
        EasyMock.verify(syncWorkflowTask, trackService);
    }

    @Test
    public void testTakeoverException1() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }

            @Override
            protected void notifySegment() throws Exception {
                throw TakeoverException.SIGNAL;
            }
        };
        providerStream.notifyProcess();
    }

    @Test(expected = WorkflowException.class)
    public void testTakeoverException2() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }

            @Override
            protected void notifySegment() throws Exception {
                throw new WorkflowException();
            }
        };
        providerStream.notifyProcess();
    }

    @Test
    public void testGetFunDataWithOutTakeover() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(100).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.isTakeover("NAME_A")).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME_B")).andReturn(false).anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMetadata()).andReturn(null).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("A", "B", "X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask1 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        SyncWorkflowTask syncWorkflowTask2 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        ProviderFunCallRequest providerFunRequest1 = ProviderFunCallRequest.builder().name("NAME_A").build();
        ProviderFunCallRequest providerFunRequest2 = ProviderFunCallRequest.builder().name("NAME_B").build();
        providerFunRequests.add(providerFunRequest1);
        providerFunRequests.add(providerFunRequest2);
        providerStream.setProviderFunRequests(providerFunRequests);
        String target = providerStream.getFunData(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("X", target);
        EasyMock.verify(request, trackService);
    }

    @Test
    public void testGetFunDataWithOneTakeover() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("A", "B", "X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask1 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        SyncWorkflowTask syncWorkflowTask2 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        ProviderFunCallRequest providerFunRequest1 = ProviderFunCallRequest.builder().name("NAME_A").build();
        ProviderFunCallRequest providerFunRequest2 = ProviderFunCallRequest.builder().name("NAME_B").build();
        providerFunRequests.add(providerFunRequest1);
        providerFunRequests.add(providerFunRequest2);
        LLMTakeover llmTakeover = new LLMTakeover();
        request.addTakeover("NAME_A", llmTakeover);
        providerStream.setProviderFunRequests(providerFunRequests);
        String target = providerStream.getFunData(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("X", target);
        EasyMock.verify(trackService);
    }

    @Test
    public void testGetFunDataWithTwoTakeover() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("A", "B", "X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {

            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }


            @Override
            protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
                Assert.assertEquals("The takeOver fun call must be the exclusive invocation of the current fun call", response);
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask1 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        SyncWorkflowTask syncWorkflowTask2 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        ProviderFunCallRequest providerFunRequest1 = ProviderFunCallRequest.builder().name("NAME_A").build();
        ProviderFunCallRequest providerFunRequest2 = ProviderFunCallRequest.builder().name("NAME_B").build();
        providerFunRequests.add(providerFunRequest1);
        providerFunRequests.add(providerFunRequest2);
        LLMTakeover llmTakeover1 = new LLMTakeover();
        LLMTakeover llmTakeover2 = new LLMTakeover();
        request.addTakeover("NAME_A", llmTakeover1);
        request.addTakeover("NAME_B", llmTakeover2);
        providerStream.setProviderFunRequests(providerFunRequests);
        try {
            providerStream.getFunData(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        } finally {
            EasyMock.verify(trackService);
        }
    }

    @Test
    public void testGetFunDataWithOneTakeoverAndOneNormal() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("A", "B", "X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {

            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }

            @Override
            // 记录FunCall Track(Request/Response)
            protected void trackFunCall(ProviderFunCallRequest request, SyncWorkflowTask response) throws Exception {
                Assert.assertEquals("B", response.get());
            }

            // 错误不存储
            @Override
            protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
                Assert.assertEquals("B", response.getResponse());
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask1 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        SyncWorkflowTask syncWorkflowTask2 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        ProviderFunCallRequest providerFunRequest1 = ProviderFunCallRequest.builder().name("NAME_A").build();
        ProviderFunCallRequest providerFunRequest2 = ProviderFunCallRequest.builder().name("NAME_B").build();
        providerFunRequests.add(providerFunRequest1);
        providerFunRequests.add(providerFunRequest2);
        LLMTakeover llmTakeover1 = new LLMTakeover();
        request.addTakeover("NAME_A", llmTakeover1);
        providerStream.setProviderFunRequests(providerFunRequests);
        providerStream.getFunData(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        EasyMock.verify(trackService);
    }

    /**
     * getFunData 中当某次 fun call 的 task.get() 抛出非静默异常时走 log.error 分支
     */
    @Test
    public void testGetFunData_nonSilentException_logErrorBranch() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andThrow(new RuntimeException("fun call failed")).anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("B").anyTimes();
        EasyMock.expect(syncWorkflowTask1.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.expect(syncWorkflowTask2.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        providerFunRequests.add(ProviderFunCallRequest.builder().name("NAME_A").build());
        providerFunRequests.add(ProviderFunCallRequest.builder().name("NAME_B").build());
        providerStream.setProviderFunRequests(providerFunRequests);
        String target = providerStream.getFunData(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("X", target);
        EasyMock.verify(trackService, syncWorkflowTask1, syncWorkflowTask2);
    }

    /**
     * getFunData 中当某次 fun call 的 task.get() 抛出 needSlient() 的 WorkflowException 时走 log.info 分支
     */
    @Test
    public void testGetFunData_silentException_logInfoBranch() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        SyncWorkflowTask syncWorkflowTask1 = EasyMock.createMock(SyncWorkflowTask.class);
        SyncWorkflowTask syncWorkflowTask2 = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.expect(syncWorkflowTask1.get()).andThrow(new WorkflowException("task closed").needSilent()).anyTimes();
        EasyMock.expect(syncWorkflowTask2.get()).andReturn("B").anyTimes();
        EasyMock.expect(syncWorkflowTask1.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.expect(syncWorkflowTask2.containFunCallTrack()).andReturn(false).anyTimes();
        EasyMock.replay(syncWorkflowTask1, syncWorkflowTask2);
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        providerFunRequests.add(ProviderFunCallRequest.builder().name("NAME_A").build());
        providerFunRequests.add(ProviderFunCallRequest.builder().name("NAME_B").build());
        providerStream.setProviderFunRequests(providerFunRequests);
        String target = providerStream.getFunData(Arrays.asList(syncWorkflowTask1, syncWorkflowTask2));
        Assert.assertEquals("X", target);
        EasyMock.verify(trackService, syncWorkflowTask1, syncWorkflowTask2);
    }

    /** storeFunCallData 中 buildFunCallData 或 historyStore.store 抛异常时走 catch，仅 log.error 不向外抛 */
    @Test
    public void testStoreFunCallData_exceptionCaught_logErrorNoThrow() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setContainHistories(true);
        request.setStoreFunCall(true);
        request.setRepositories(Arrays.asList("UNKNOWN"));
        request.setExpired(1000);
        request.setHistories(100);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyObject(HistoryPair.class), EasyMock.anyObject(), EasyMock.anyObject());
        EasyMock.expectLastCall().andThrow(new RuntimeException("store failed")).once();
        EasyMock.replay(historyStore);
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("B", "X");
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierService)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }
            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
            @Override
            protected HistoryPair buildFunCallData(ProviderFunCallRequest req, ProviderFunCallResponse res) throws Exception {
                HistoryPair pair = new HistoryPair(this.request.getMessage(), req.getCreated());
                pair.setAnswer(JsonUtils.write(res));
                pair.setQuery(JsonUtils.write(req));
                pair.setFunction(History.FUN_FUNCALL);
                return pair;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask1 = SyncWorkflowTask.exeWorkflow(notifierService, SyncConfig.builder().workTask(workflowTask).build());
        List<ProviderFunCallRequest> providerFunRequests = new ArrayList<>();
        providerFunRequests.add(ProviderFunCallRequest.builder().name("NAME_A").build());
        providerStream.setProviderFunRequests(providerFunRequests);
        String target = providerStream.getFunData(Arrays.asList(syncWorkflowTask1));
        Assert.assertEquals("X", target);
        EasyMock.verify(trackService, historyStore);
    }

    @Test
    public void testFunMetadata1() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(100).anyTimes();
        EasyMock.expect(request.isTakeover("NAME_A")).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME_B")).andReturn(false).anyTimes();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of(ProviderRequestService.KEY_FUN_MERGE, "F0", ProviderRequestService.KEY_FUN_FETCH, "F1", "HELLO", "WORLD"));
        MessageDelegate messageDelegate = new MessageDelegate(llmQuery);
        llmQuery.setWorkflow(ProviderRequestService.KEY_FUN_SELECT);
        EasyMock.expect(request.getMessage()).andReturn(messageDelegate).anyTimes();
        EasyMock.expect(request.getMetadata()).andReturn(llmQuery.getMetadata()).anyTimes();
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService, request);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        Map<String, Object> metadata = providerStream.getFunMetadata();
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(metadata.size()));
        Assert.assertEquals("WORLD", metadata.get("HELLO"));
        EasyMock.verify(trackService, request);
    }

    @Test
    public void testFunMetadata2() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getFunCallTimeout()).andReturn(100).anyTimes();
        EasyMock.expect(request.isTakeover("NAME_A")).andReturn(false).anyTimes();
        EasyMock.expect(request.isTakeover("NAME_B")).andReturn(false).anyTimes();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of(ProviderRequestService.KEY_FUN_MERGE, "F0", ProviderRequestService.KEY_FUN_FETCH, "F1", "HELLO", "WORLD"));
        MessageDelegate messageDelegate = new MessageDelegate(llmQuery);
        llmQuery.setWorkflow(ProviderRequestService.KEY_FUN_SELECT);
        EasyMock.expect(request.getMessage()).andReturn(messageDelegate).anyTimes();
        EasyMock.expect(request.getMetadata()).andReturn(llmQuery.getMetadata()).anyTimes();
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService, request);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        Map<String, Object> metadata = providerStream.getFunMetadata("X", "Y");
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(metadata.size()));
        Assert.assertEquals("WORLD", metadata.get("HELLO"));
        Assert.assertEquals("Y", metadata.get("X"));
        EasyMock.verify(trackService, request);
    }

    @Test
    public void testStoreHistory1() throws Exception {
        ProviderRequest provider = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(provider.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(provider.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(provider.getStoreCompleted()).andReturn(true).anyTimes();
        EasyMock.expect(provider.getRepositories()).andReturn(List.of("A", "B")).anyTimes();
        EasyMock.expect(provider.getQuery4History()).andReturn("QUERY").anyTimes();
        EasyMock.expect(provider.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(provider.getHistories()).andReturn(10086).anyTimes();
        EasyMock.expect(provider.getExpired()).andReturn(10086).anyTimes();
        EasyMock.expect(provider.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(provider.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(provider.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(provider.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(provider.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(provider.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(provider);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        List<HistoryPair> mockHistories = new ArrayList<>();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(provider.getMessage(), provider.getRepositories(), mockHistories, provider.getExpired(), provider.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackService, historyStore);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(provider)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals("QUERY", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(provider.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                Assert.assertTrue(historyPairs.getLast().getAnswer().contains("HELLO"));
                Assert.assertTrue(Long.valueOf(provider.getMessage().getCreated()) <= Long.valueOf(historyPairs.getLast().getCreated()));
                return mockHistories;
            }
        };
        providerStream.storeConversation("HELLO");
        EasyMock.verify(provider, trackService, historyStore);
    }

    @Test
    public void testStoreHistoryWithFunCall() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getRepositories()).andReturn(List.of("A", "B")).anyTimes();
        EasyMock.expect(request.getQuery4History()).andReturn("QUERY").anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(10086).anyTimes();
        EasyMock.expect(request.getExpired()).andReturn(10086).anyTimes();
        EasyMock.expect(request.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.putMetadata(ProviderRequestService.KEY_FUN_FETCH, "HELLO");
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        // historyStore.store(request.getMessage(), request.getRepositories(), request.getQuery4History(), "HELLO", request.getExpired(), request.getHistories(), request.getMessage().getTimestamp());
        // EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackService, historyStore);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        providerStream.storeConversation("HELLO");
        EasyMock.verify(request, trackService, historyStore);
    }

    @Test
    public void testStoreHistoryWithFunCallAndStoreFunCall() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getRepositories()).andReturn(List.of("A", "B")).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getQuery4History()).andReturn("QUERY").anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(10086).anyTimes();
        EasyMock.expect(request.getExpired()).andReturn(10086).anyTimes();
        EasyMock.expect(request.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.putMetadata(ProviderRequestService.KEY_FUN_FETCH, "HELLO");
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(request.getMessage(), request.getRepositories(), request.getQuery4History(), "HELLO", request.getExpired(), request.getHistories(), request.getMessage().getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackService, historyStore);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        providerStream.storeConversation("HELLO");
        EasyMock.verify(request, trackService, historyStore);
    }

    @Test
    public void testBuildFunCallData() throws Exception {
        NamesServiceImpl namesService = NamesServiceImpl.class.cast(ObjectBuilder.buildNamesService());
        namesService.setEncode(false);
        String name = namesService.encode(NamesService.PREFIX_TOOLS, "abc", "fn");
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getRepositories()).andReturn(List.of("A", "B")).anyTimes();
        EasyMock.expect(request.getQuery4History()).andReturn("QUERY").anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(true).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(10086).anyTimes();
        EasyMock.expect(request.getExpired()).andReturn(10086).anyTimes();
        EasyMock.expect(request.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.putMetadata(ProviderRequestService.KEY_FUN_FETCH, "HELLO");
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(request.getMessage(), request.getRepositories(), request.getQuery4History(), "HELLO", request.getExpired(), request.getHistories(), 1L);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackService, historyStore);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(namesService)
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder().name(name).refer(ImmutableMap.of("A", "B")).build();
        HistoryPair pairs = providerStream.buildFunCallData(providerFunCallRequest, ProviderFunCallResponse.builder().created(1L).name(name).response("WORLD").build());
        Assert.assertEquals("{\"A\":\"B\"}", JsonUtils.write(JsonUtils.read(pairs.getQuery(), Map.class).get("refer")));
        Assert.assertEquals("{\"created\":1,\"response\":\"WORLD\",\"name\":\"Tools_abc__fn\",\"valid\":true}", pairs.getAnswer());
        Assert.assertNotNull(pairs.getCreated());
        EasyMock.verify(request, trackService, historyStore);
    }

    @Test
    public void testBuildFunCallData_allFields() throws Exception {
        NamesServiceImpl namesService = NamesServiceImpl.class.cast(ObjectBuilder.buildNamesService());
        namesService.setEncode(false);
        String name = namesService.encode(NamesService.PREFIX_TOOLS, "abc", "fn");
        String workflow = "WF";
        String biz = "BIZ";
        String chat = "chat-002";
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow(workflow);
        message.setBiz(biz);
        message.setChat(chat);
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(namesService)
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        ProviderFunCallRequest funCallRequest = ProviderFunCallRequest.builder().name(name).refer(ImmutableMap.of("k", "v")).build();
        ProviderFunCallResponse funCallResponse = ProviderFunCallResponse.builder().response("resp").name(name).build();
        HistoryPair pair = providerStream.buildFunCallData(funCallRequest, funCallResponse);
        Assert.assertNotNull(pair);
        Assert.assertEquals(funCallRequest.getCreated(), pair.getCreated());
        Assert.assertEquals("abc@fn", pair.getSource());
        Assert.assertEquals(message.getConversation(), pair.getConversation());
        Assert.assertEquals(chat, pair.getChat());
        Assert.assertEquals(History.FUN_FUNCALL, pair.getFunction());
        Assert.assertEquals(JsonUtils.write(funCallRequest), pair.getQuery());
        Assert.assertEquals(JsonUtils.write(funCallResponse), pair.getAnswer());
    }

    /** buildFunCallData：request/response 上的 model 写入 HistoryPair 的 query/answer JSON（与 store 序列化一致）。 */
    @Test
    public void testBuildFunCallData_includesModelInSerializedQueryAndAnswer() throws Exception {
        NamesServiceImpl namesService = NamesServiceImpl.class.cast(ObjectBuilder.buildNamesService());
        namesService.setEncode(false);
        String name = namesService.encode(NamesService.PREFIX_TOOLS, "abc", "fn");
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(namesService)
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        ProviderFunCallRequest funCallRequest = ProviderFunCallRequest.builder()
                .name(name)
                .model("stream-model-id")
                .api("fun-api")
                .refer(com.google.common.collect.ImmutableMap.of("k", "v"))
                .build();
        ProviderFunCallResponse funCallResponse = ProviderFunCallResponse.builder()
                .response("resp")
                .name(name)
                .model("response-model-id")
                .build();
        HistoryPair pair = providerStream.buildFunCallData(funCallRequest, funCallResponse);
        Assert.assertEquals(JsonUtils.write(funCallRequest), pair.getQuery());
        Assert.assertEquals(JsonUtils.write(funCallResponse), pair.getAnswer());
        Assert.assertTrue(pair.getQuery().contains("stream-model-id"));
        Assert.assertTrue(pair.getAnswer().contains("response-model-id"));
        Assert.assertEquals("stream-model-id", pair.getModel());
        Assert.assertEquals("fun-api", pair.getApi());
    }

    @Test
    public void testBuildConversationRequest() throws Exception {
        String query4History = "query-for-history";
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setChat("chat-001");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        request.setPureQuery(false);
        request.setModel("conv-req-model");
        request.setApi("google");
        message.setQuery(query4History);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        HistoryPair pair = providerStream.buildConversationRequest();
        Assert.assertNotNull(pair);
        Assert.assertEquals(Long.valueOf(message.getCreated() + 1), pair.getCreated());
        Assert.assertEquals(message.getConversation(), pair.getConversation());
        Assert.assertEquals("chat-001", pair.getChat());
        Assert.assertEquals(query4History, pair.getQuery());
        Assert.assertNull(pair.getAnswer());
        Assert.assertEquals("conv-req-model", pair.getModel());
        Assert.assertEquals("google", pair.getApi());
    }

    @Test
    public void testBuildConversationResponse() throws Exception {
        String content = "answer-content";
        String query4History = "query-4-history";
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setChat("chat-resp");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        request.setModel("resp-model");
        request.setApi("google");
        message.setQuery(query4History);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        HistoryPair pair = providerStream.buildConversationResponse(content);
        Assert.assertNotNull(pair);
        Assert.assertEquals("chat-resp", pair.getChat());
        Assert.assertEquals(null, pair.getQuery());
        Assert.assertEquals(content, pair.getAnswer());
        Assert.assertNull(pair.getReasoning());
        Assert.assertEquals("resp-model", pair.getModel());
        Assert.assertEquals("google", pair.getApi());
        // 使用当前时间：created 应接近调用时的时间戳
        long now = System.currentTimeMillis();
        Assert.assertNotNull(pair.getCreated());
        Assert.assertTrue("created should be current time", Math.abs(pair.getCreated() - now) < 2000L);
    }

    @Test
    public void testBuildConversationResponse_withReasoning() throws Exception {
        String content = "answer-content";
        String reasoningText = "reasoning-text";
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setChat("chat-r");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        request.setPrintReason(true);
        message.setQuery("q");
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }
        };
        Field reasoningField = ProviderStream.class.getDeclaredField("reasoning");
        reasoningField.setAccessible(true);
        reasoningField.set(providerStream, new StringBuffer(reasoningText));
        HistoryPair pair = providerStream.buildConversationResponse(content);
        Assert.assertNotNull(pair);
        Assert.assertEquals(content, pair.getAnswer());
        Assert.assertEquals(reasoningText, pair.getReasoning());
        Assert.assertNotNull(pair.getCreated());
    }

    /**
     * reasoning 有内容但 printReason=false 时，response 的 reasoning 不写入（shouldStoreReasoning 为 false）
     */
    @Test
    public void testBuildConversationResponse_reasoningNotStoredWhenPrintReasonFalse() throws Exception {
        String content = "answer";
        String reasoningText = "some-reasoning";
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setChat("ch");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        request.setPrintReason(false);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return null;
            }

            @Override
            protected Boolean atonce(String source) {
                return null;
            }
        };
        Field reasoningField = ProviderStream.class.getDeclaredField("reasoning");
        reasoningField.setAccessible(true);
        reasoningField.set(providerStream, new StringBuffer(reasoningText));
        HistoryPair pair = providerStream.buildConversationResponse(content);
        Assert.assertNotNull(pair);
        Assert.assertEquals(content, pair.getAnswer());
        Assert.assertNull("reasoning should not be stored when printReason is false", pair.getReasoning());
    }

    @Test
    public void testNotifyFirstToken() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getTokenFirst()).andReturn(5).anyTimes();
        EasyMock.expect(request.getTokenBuffer()).andReturn(100).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();

        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.signal(EasyMock.anyObject(), EasyMock.eq(message));
        EasyMock.expectLastCall().once();

        NotifierServiceImpl notifierService = EasyMock.createMock(NotifierServiceImpl.class);
        notifierService.notify(EasyMock.anyObject(), EasyMock.eq(message), EasyMock.eq(message));
        EasyMock.expectLastCall().once();

        EasyMock.replay(request, signal, notifierService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(notifierService)
                .providerReason(null)
                .signalStream(signal)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.contentBuffer.append("123456"); // Length 6 > tokenFirst 5
        stream.notify(0, false);
        EasyMock.verify(request, signal, notifierService);
    }

    @Test
    public void testNotifyBufferReady() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getTokenFirst()).andReturn(100).anyTimes();
        EasyMock.expect(request.getTokenBuffer()).andReturn(5).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();

        SignalStream signal = EasyMock.createMock(SignalStream.class);
        signal.signal(EasyMock.anyObject(), EasyMock.eq(message));
        EasyMock.expectLastCall().once();

        NotifierServiceImpl notifierService = EasyMock.createMock(NotifierServiceImpl.class);
        notifierService.notify(EasyMock.anyObject(), EasyMock.eq(message), EasyMock.eq(message));
        EasyMock.expectLastCall().once();

        EasyMock.replay(request, signal, notifierService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(notifierService)
                .providerReason(null)
                .signalStream(signal)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.offset = 0;
        stream.contentBuffer.append("123456"); // Length 6 - offset 0 > tokenBuffer 5
        stream.notify(1, false);
        EasyMock.verify(request, signal, notifierService);
    }

    @Test(expected = TakeoverException.class)
    public void testGetFunDataTakeoverSignal() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.isTakeover(EasyMock.anyString())).andReturn(true).anyTimes();
        LLMTakeover takeover = new LLMTakeover();
        EasyMock.expect(request.getTakeover(EasyMock.anyString())).andReturn(takeover).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();

        EasyMock.replay(request);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.providerFunRequests = Arrays.asList(ProviderFunCallRequest.builder().name("T1").build());
        SyncWorkflowTask task = EasyMock.createMock(SyncWorkflowTask.class);
        EasyMock.replay(task);

        stream.getFunData(Arrays.asList(task));
    }

    @Test
    public void testStoreConversationNotWriteable() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.isWriteable()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();

        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(request, historyStore);

        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.storeConversation("CONTENT");
        EasyMock.verify(historyStore); // store should not be called
    }

    @Test
    public void testStoreConversationFromFunCall() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getNotifier(Notifier.ENDPOINT)).andReturn(Notifier.ENDPOINT).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.isWriteable()).andReturn(true).anyTimes();

        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.putMetadata(ProviderRequestService.KEY_FUN_FETCH, "TRUE");
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();

        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(request, historyStore);

        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };
        stream.storeConversation("CONTENT");
        EasyMock.verify(historyStore); // store should not be called
    }

    @Test
    public void testStoreCompleted() throws Exception {
        ProviderRequest provider = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(provider.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(provider.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(provider.getStoreCompleted()).andReturn(false).anyTimes();
        EasyMock.expect(provider.getRepositories()).andReturn(List.of("A", "B")).anyTimes();
        EasyMock.expect(provider.getQuery4History()).andReturn("QUERY").anyTimes();
        EasyMock.expect(provider.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(provider.getHistories()).andReturn(10086).anyTimes();
        EasyMock.expect(provider.getExpired()).andReturn(10086).anyTimes();
        EasyMock.expect(provider.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(provider.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(provider.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(provider.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(provider.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(provider.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(provider);
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        List<HistoryPair> mockHistories = new ArrayList<>();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        historyStore.store(provider.getMessage(), provider.getRepositories(), mockHistories, provider.getExpired(), provider.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackService, historyStore);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(historyStore)
                .namesService(ObjectBuilder.buildNamesService())
                .request(provider)
                .build()) {
            @Override
            protected Boolean stream(String source) throws Exception {
                return null;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return null;
            }

            @Override
            protected List<HistoryPair> buildConversationHistories(String content) throws Exception {
                List<HistoryPair> historyPairs = super.buildConversationHistories(content);
                Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(historyPairs.size()));
                Assert.assertTrue(historyPairs.getFirst().getAnswer().contains("HELLO"));
                Assert.assertTrue(Long.valueOf(provider.getMessage().getCreated()) <= Long.valueOf(historyPairs.getFirst().getCreated()));
                return mockHistories;
            }
        };
        providerStream.storeConversation("HELLO");
        EasyMock.verify(provider, trackService, historyStore);
    }

    /**
     * 覆盖 updateConversation：空列表，循环不执行，直接返回。
     */
    @Test
    public void testUpdateConversation_emptyList() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);

        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };

        List<HistoryPair> emptyList = new ArrayList<>();
        List<HistoryPair> result = stream.updateConversation(emptyList);
        Assert.assertSame(emptyList, result);
        Assert.assertEquals(0, result.size());
        EasyMock.verify(request);
    }

    /**
     * 覆盖 updateConversation：单个元素，循环执行一次，仅设置 function 为 FUN_CHAT。
     */
    @Test
    public void testUpdateConversation_singleElement() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);

        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };

        HistoryPair pair = new HistoryPair();
        pair.setQuery("query");
        pair.setAnswer("answer");
        List<HistoryPair> pairs = new ArrayList<>();
        pairs.add(pair);

        List<HistoryPair> result = stream.updateConversation(pairs);
        Assert.assertSame(pairs, result);
        Assert.assertEquals(1, result.size());
        Assert.assertEquals(History.FUN_CHAT, result.get(0).getFunction());
        EasyMock.verify(request);
    }

    /**
     * 覆盖 updateConversation：多个元素，循环执行多次，每个都设置 function 为 FUN_CHAT。
     */
    @Test
    public void testUpdateConversation_multipleElements() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.replay(request);

        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(null)
                .mediaInlineService(null)
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return true;
            }

            @Override
            protected Boolean atonce(String source) {
                return true;
            }
        };

        HistoryPair pair1 = new HistoryPair();
        pair1.setQuery("query1");
        HistoryPair pair2 = new HistoryPair();
        pair2.setQuery("query2");
        HistoryPair pair3 = new HistoryPair();
        pair3.setQuery("query3");
        List<HistoryPair> pairs = new ArrayList<>();
        pairs.add(pair1);
        pairs.add(pair2);
        pairs.add(pair3);

        List<HistoryPair> result = stream.updateConversation(pairs);
        Assert.assertSame(pairs, result);
        Assert.assertEquals(3, result.size());
        for (HistoryPair p : result) {
            Assert.assertEquals(History.FUN_CHAT, p.getFunction());
        }
        EasyMock.verify(request);
    }

    @Test
    public void testBuildFailedMessage() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return null;
            }

            @Override
            protected Boolean atonce(String source) {
                return null;
            }
        };
        java.lang.reflect.Method m = ProviderStream.class.getDeclaredMethod("buildFailedMessage", Exception.class);
        m.setAccessible(true);
        Exception e = new Exception("test-error-detail");
        String msg = (String) m.invoke(stream, e);
        String prefix = "The error occurred, details are as follows: ";
        String expected = prefix + "`" + "test-error-detail" + "`, please refrain from retrying similar errors.";
        Assert.assertEquals(expected, msg);
        EasyMock.verify(trackService);
    }

    /**
     * 当 e.getMessage() 以 prefix 开头时，中间段为 e.getMessage()，否则为 prefix + e.getMessage()。
     */
    @Test
    public void testBuildFailedMessageWhenExceptionMessageStartsWithPrefix() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return null;
            }

            @Override
            protected Boolean atonce(String source) {
                return null;
            }
        };
        java.lang.reflect.Method m = ProviderStream.class.getDeclaredMethod("buildFailedMessage", Exception.class);
        m.setAccessible(true);
        String prefix = "The error occurred, details are as follows: ";
        String messageContent = prefix + "something wrong";
        Exception e = new Exception(messageContent);
        String msg = (String) m.invoke(stream, e);
        Assert.assertEquals(msg, messageContent);
        EasyMock.verify(trackService);
    }

    /**
     * 当 e.getMessage() 为 null 时，中间段为 prefix + null，结尾为 ", please refrain from retrying similar errors."。
     */
    @Test
    public void testBuildFailedMessageWhenExceptionMessageIsNull() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService trackService = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(trackService);
        ProviderStream stream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(trackService)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(null)
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected Boolean stream(String source) {
                return null;
            }

            @Override
            protected Boolean atonce(String source) {
                return null;
            }
        };
        java.lang.reflect.Method m = ProviderStream.class.getDeclaredMethod("buildFailedMessage", Exception.class);
        m.setAccessible(true);
        Exception e = new Exception((String) null);
        String msg = (String) m.invoke(stream, e);
        String prefix = "The error occurred, details are as follows: ";
        String expected = prefix + "`" + "null" + "`, please refrain from retrying similar errors.";
        Assert.assertEquals(expected, msg);
        EasyMock.verify(trackService);
    }
}
