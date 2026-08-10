package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import ai.open.right.workflow.flow.llm.signal.SignalDistributor;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class SignalStreamImplTest {

    @Test
    public void testFinish() throws Exception {
        List<String> synthesizer = new ArrayList<String>();
        synthesizer.add("HELLO");
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalConfig config = new SignalConfig();
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        signalDistributor.distribute(config, synthesizer, message);
        EasyMock.expectLastCall().anyTimes();
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        signalStream.setSynthesizer(synthesizer);
        EasyMock.replay(signalDistributor);
        signalStream.finish(message);
        EasyMock.verify(signalDistributor);
        Assert.assertNotNull(signalStream.getSignalConfig());
        Assert.assertNotNull(signalStream.getSignalDistributor());
        Assert.assertNotNull(signalStream.getSynthesizer());
    }

    @Test
    public void testRemove() throws Exception {
        SignalConfig config = new SignalConfig();
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        EasyMock.replay(signalDistributor);
        Assert.assertEquals(signalStream.remove("HELLO ${WO}RLD"), "HELLO RLD");
        EasyMock.verify(signalDistributor);
    }

    @Test
    public void testSignal1() throws Exception {
        SignalConfig config = new SignalConfig();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        signalDistributor.distribute(config, "KEY=VAL1", message);
        EasyMock.expectLastCall().times(1);
        signalDistributor.distribute(config, "KEY2=VAL2", message);
        EasyMock.expectLastCall().times(1);
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(providerRequest.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getChain()).andReturn("NEXT").anyTimes();
        EasyMock.expect(providerRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(providerRequest.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(signalDistributor, providerRequest);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signalStream)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(providerRequest)
                .build()) {
            protected Boolean stream(String source) throws Exception {
                return false;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return false;
            }
        };
        providerStream.getContentBuffer().append("HELLO WORLD ${KEY=VAL1} HELLO WORLD ${KEY2=VAL2}");
        signalStream.signal(providerStream, message);
        Assert.assertEquals("HELLO WORLD  HELLO WORLD ", providerStream.getContentBuffer().toString());
        EasyMock.verify(signalDistributor, providerRequest, t);
    }

    @Test
    public void testSignal2() throws Exception {
        SignalConfig config = new SignalConfig();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        signalDistributor.distribute(config, "KEY=VAL1", message);
        EasyMock.expectLastCall().times(1);
        signalDistributor.distribute(config, "KEY2=VAL2", message);
        EasyMock.expectLastCall().times(1);
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(providerRequest.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getChain()).andReturn("NEXT").anyTimes();
        EasyMock.expect(providerRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(providerRequest.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(signalDistributor, providerRequest);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signalStream)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(providerRequest)
                .build()) {
            protected Boolean stream(String source) throws Exception {
                return false;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return false;
            }
        };
        providerStream.getContentBuffer().append("${KEY=VAL1} HELLO WORLD ${KEY2=VAL2}");
        signalStream.signal(providerStream, message);
        Assert.assertEquals(" HELLO WORLD ", providerStream.getContentBuffer().toString());
        EasyMock.verify(signalDistributor, providerRequest, t);
    }

    @Test
    public void testSignal3() throws Exception {
        SignalConfig config = new SignalConfig();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        signalDistributor.distribute(config, "KEY=VAL1", message);
        EasyMock.expectLastCall().times(1);
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(providerRequest.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getChain()).andReturn("NEXT").anyTimes();
        EasyMock.expect(providerRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(providerRequest.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(signalDistributor, providerRequest);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signalStream)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(providerRequest)
                .build()) {
            protected Boolean stream(String source) throws Exception {
                return false;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return false;
            }
        };
        providerStream.getContentBuffer().append("${KEY=VAL1} HELLO WORLD ${KEY2");
        signalStream.signal(providerStream, message);
        Assert.assertEquals(" HELLO WORLD ${KEY2", providerStream.getContentBuffer().toString());
        EasyMock.verify(signalDistributor, providerRequest, t);
    }

    @Test
    public void testSignal4() throws Exception {
        SignalConfig config = new SignalConfig();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(providerRequest.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getChain()).andReturn("NEXT").anyTimes();
        EasyMock.expect(providerRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(providerRequest.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(signalDistributor, providerRequest);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signalStream)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(providerRequest)
                .build()) {
            protected Boolean stream(String source) throws Exception {
                return false;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return false;
            }
        };
        providerStream.getContentBuffer().append("${KEY=VAL1 HELLO WORLD ${KEY2");
        signalStream.signal(providerStream, message);
        Assert.assertEquals("${KEY=VAL1 HELLO WORLD ${KEY2", providerStream.getContentBuffer().toString());
        EasyMock.verify(signalDistributor, providerRequest, t);
    }

    @Test
    public void testSignal4WithSynthesizer() throws Exception {
        SignalConfig config = new SignalConfig();
        config.setSynthesizer("NEXT");
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalDistributor signalDistributor = EasyMock.createMock(SignalDistributor.class);
        SignalStreamImpl signalStream = new SignalStreamImpl(config, signalDistributor);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(providerRequest.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(providerRequest.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getChain()).andReturn("NEXT").anyTimes();
        EasyMock.expect(providerRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(providerRequest.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(providerRequest.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(signalDistributor, providerRequest);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        ProviderStream providerStream = new ProviderStream(ProviderStreamConfig.<ProviderRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(signalStream)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(providerRequest)
                .build()) {
            protected Boolean stream(String source) throws Exception {
                return false;
            }

            @Override
            protected Boolean atonce(String source) throws Exception {
                return false;
            }
        };
        providerStream.getContentBuffer().append("${KEY=VAL1} HELLO WORLD ${KEY2=VAL2}");
        signalStream.signal(providerStream, message);
        Assert.assertEquals(" HELLO WORLD ", providerStream.getContentBuffer().toString());
        EasyMock.verify(signalDistributor, providerRequest, t);
    }
}