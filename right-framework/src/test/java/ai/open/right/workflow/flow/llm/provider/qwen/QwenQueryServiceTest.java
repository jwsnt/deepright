package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class QwenQueryServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = QwenQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = QwenQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testInit() {
        QwenRequestService qwen = EasyMock.createMock(QwenRequestService.class);
        QwenRouter router = EasyMock.createMock(QwenRouter.class);
        EasyMock.replay(qwen, router);
        QwenQueryService service = new QwenQueryService();
        service.setQwenRequestService(qwen);
        service.setQwenRouter(router);
        Assert.assertEquals(qwen, service.request());
        Assert.assertEquals(router, service.router());
        EasyMock.verify(qwen, router);
    }

    @Test
    public void testStream() throws Exception {
        QwenRequestService qwen = EasyMock.createMock(QwenRequestService.class);
        OpenAiRequest request = EasyMock.createMock(OpenAiRequest.class);
        QwenRouter router = EasyMock.createMock(QwenRouter.class);
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        NotifierServiceImpl notifier = EasyMock.createMock(NotifierServiceImpl.class);
        HistoryStore history = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(qwen, router, signal, request, history);
        QwenQueryService service = new QwenQueryService();
        service.setHistoryStore(history);
        service.setNotifierService(notifier);
        service.setQwenRequestService(qwen);
        service.setQwenRouter(router);
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNamesService(new NamesServiceImpl());
        OpenAiStream stream = service.stream(signal, request);
        Assert.assertEquals(stream.getHistoryStore(), history);
        Assert.assertEquals(stream.getRequest(), request);
        Assert.assertEquals(stream.getNotifierService(), notifier);
        Assert.assertEquals(stream.getSignalStream(), signal);
        EasyMock.verify(qwen, router, signal, request, history);
    }
}
