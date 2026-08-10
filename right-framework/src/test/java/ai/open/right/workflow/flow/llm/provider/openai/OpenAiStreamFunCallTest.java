package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class OpenAiStreamFunCallTest {

    @Test
    public void testAfter() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        OpenAiRequest c = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(null).anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.isTakeover("Tools_shell__check_file_exist")).andReturn(false).anyTimes();
        EasyMock.expect(c.getTakeover("Tools_shell__check_file_exist")).andReturn(null).anyTimes();
        EasyMock.expect(c.isTakeover("TOOLS_MY_TOOLS")).andReturn(false).anyTimes();
        EasyMock.expect(c.getTakeover("TOOLS_MY_TOOLS")).andReturn(null).anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(1000).anyTimes();
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
        h.store(c.getMessage(), c.getRepositories(), c.getQuery4History(), "CONTENT", "A", c.getExpired(), c.getHistories(), c.getMessage().getCreated());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        OpenAiStreamFunCall stream = new OpenAiStreamFunCall(ProviderStreamConfig.<OpenAiRequest>builder()
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
            protected void notifySegment() throws Exception {
                Assert.assertEquals("HELLO WORLD HELLO", this.segment.getContent().toString());
                Assert.assertTrue(this.segment.isFinished());
            }

            @Override
            protected void notify(int seqid, boolean finished) throws Exception {

            }
        };
        stream.afterCreateFunRequest(null, null, null, null, null, null);
        stream.afterUpdateFunRequest(null, null, null, null, null, null);
    }
}
