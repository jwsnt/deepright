package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderData;
import ai.open.right.workflow.flow.llm.provider.ProviderUtils;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
public class OpenAiReaderTest {

    @Test
    public void test() throws Exception {
        OpenAiRequest req = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(req.hasAutoDump()).andReturn(false).anyTimes();
        req.autoDump();
        EasyMock.expectLastCall().anyTimes();
        req.appendResponse("data: [DONE]");
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(true).anyTimes();
        LLMCallback cal = EasyMock.createMock(LLMCallback.class);
        cal.callback(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(req, cal);
        AtomicBoolean finished = new AtomicBoolean(false);
        StringBuilder builder = new StringBuilder();
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(req)
                .llmCallback(cal)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {
            // 追加消息
            @Override
            protected void completed(String message) throws Exception {
                super.completed(message);
                builder.append(message);
                finished.set(true);
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, "data: [DONE]"), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        Assert.assertEquals("data: [DONE]", builder.toString());
        EasyMock.verify(req, cal);
    }

    @Test
    public void testWithDone() throws Exception {
        OpenAiRequest req = EasyMock.createMock(OpenAiRequest.class);
        req.appendResponse("data: [DONE]");
        EasyMock.expectLastCall().anyTimes();
        req.autoDump();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(req.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(req.hasChain()).andReturn(false).anyTimes();
        EasyMock.replay(req);
        AtomicBoolean finished = new AtomicBoolean(false);
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }
        };
        OpenAiStream stream = new OpenAiStream(ProviderStreamConfig.<OpenAiRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(notifierManager)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(req)
                .build()) {
        };
        StringBuilder builder = new StringBuilder();
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(req)
                .llmCallback(stream)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                super.completed(message);
                builder.append(message);
                finished.set(true);
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, "data: [DONE]"), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        // data: [DONE] 会被过滤
        Assert.assertEquals("data: [DONE]", builder.toString());
        EasyMock.verify(req);
    }

    @Test
    public void testWithOnce() throws Exception {
        OpenAiRequest req = EasyMock.createMock(OpenAiRequest.class);
        req.appendResponse("data: [DONE]");
        EasyMock.expectLastCall().anyTimes();
        req.autoDump();
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(req.getProviderData()).andReturn(new ProviderData()).anyTimes();
        EasyMock.expect(req.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(req.getStream()).andReturn(false).anyTimes();
        LLMCallback cal = EasyMock.createMock(LLMCallback.class);
        cal.callback(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(req, cal);
        AtomicBoolean finished = new AtomicBoolean(false);
        StringBuilder builder = new StringBuilder();
        OpenAiReader reader = new OpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .request(req)
                .llmCallback(cal)
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect())
                .eventListenerService(ObjectBuilder.buildEventListenerService())
                .extension(new HashMap<>())
                .discard(1024)
                .timeout(1024)
                .buffer(1024)
                .capacity(1024)
                .queue(1024)
                .build()) {

            @Override
            protected void completed(String message) throws Exception {
                super.completed(message);
                builder.append(message);
                finished.set(true);
            }
        };
        reader.consuming(Executors.newFixedThreadPool(1));
        reader.consumeContent(ProviderUtils.buildDecoder(reader, "data: [DONE]"), null);
        ProviderUtils.buildResult(reader);
        while (!finished.get()) {
        }
        Assert.assertEquals("data: [DONE]", builder.toString());
        EasyMock.verify(req, cal);
    }
}
