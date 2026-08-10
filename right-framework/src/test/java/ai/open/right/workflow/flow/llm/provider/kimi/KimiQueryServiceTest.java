package ai.open.right.workflow.flow.llm.provider.kimi;

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

public class KimiQueryServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = KimiQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = KimiQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testInit() {
        KimiRequestService kimi = EasyMock.createMock(KimiRequestService.class);
        KimiRouter router = EasyMock.createMock(KimiRouter.class);
        EasyMock.replay(kimi, router);
        KimiQueryService service = new KimiQueryService();
        MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl();
        service.setMediaInlineService(mediaInlineService);
        service.setKimiRequestService(kimi);
        service.setKimiRouter(router);
        Assert.assertEquals(mediaInlineService, service.getMediaInlineService());
        Assert.assertEquals(kimi, service.request());
        Assert.assertEquals(router, service.router());
        EasyMock.verify(kimi, router);
    }

    @Test
    public void testStream() throws Exception {
        KimiRequestService kimi = EasyMock.createMock(KimiRequestService.class);
        OpenAiRequest request = EasyMock.createMock(OpenAiRequest.class);
        KimiRouter router = EasyMock.createMock(KimiRouter.class);
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        NotifierServiceImpl notifier = EasyMock.createMock(NotifierServiceImpl.class);
        HistoryStore history = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(kimi, router, signal, request, history);
        KimiQueryService service = new KimiQueryService();
        service.setHistoryStore(history);
        service.setNotifierService(notifier);
        service.setKimiRequestService(kimi);
        service.setKimiRouter(router);
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNamesService(new NamesServiceImpl());
        OpenAiStream stream = service.stream(signal, request);
        Assert.assertEquals(stream.getHistoryStore(), history);
        Assert.assertEquals(stream.getRequest(), request);
        Assert.assertEquals(stream.getNotifierService(), notifier);
        Assert.assertEquals(stream.getSignalStream(), signal);
        EasyMock.verify(kimi, router, signal, request, history);
    }
}
