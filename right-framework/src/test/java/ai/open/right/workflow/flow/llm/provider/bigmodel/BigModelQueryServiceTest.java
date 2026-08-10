package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.store.history.impl.RedisHistoryStore;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class BigModelQueryServiceTest {

    @Test
    public void testInit() {
        BigModelRequestService deepseek = EasyMock.createMock(BigModelRequestService.class);
        BigModelRouter router = EasyMock.createMock(BigModelRouter.class);
        EasyMock.replay(deepseek, router);
        BigModelQueryService service = new BigModelQueryService();
        service.setBigModelRequestService(deepseek);
        service.setBigModelRouter(router);
        Assert.assertEquals(deepseek, service.request());
        Assert.assertEquals(router, service.router());
        EasyMock.verify(deepseek, router);
    }

    @Test
    public void testStream() throws Exception {
        BigModelRequestService deepseek = EasyMock.createMock(BigModelRequestService.class);
        OpenAiRequest request = EasyMock.createMock(OpenAiRequest.class);
        BigModelRouter router = EasyMock.createMock(BigModelRouter.class);
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        NotifierServiceImpl notifier = EasyMock.createMock(NotifierServiceImpl.class);
        HistoryStore history = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(deepseek, router, signal, request, history);
        BigModelQueryService service = new BigModelQueryService();
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        service.setNamesService(new NamesServiceImpl());
        service.setHistoryStore(history);
        service.setNotifierService(notifier);
        service.setBigModelRequestService(deepseek);
        service.setBigModelRouter(router);
        OpenAiStream stream = service.stream(signal, request);
        Assert.assertEquals(stream.getHistoryStore(), history);
        Assert.assertEquals(stream.getRequest(), request);
        Assert.assertEquals(stream.getNotifierService(), notifier);
        Assert.assertEquals(stream.getSignalStream(), signal);
        EasyMock.verify(deepseek, router, signal, request, history);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = BigModelQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = BigModelQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
