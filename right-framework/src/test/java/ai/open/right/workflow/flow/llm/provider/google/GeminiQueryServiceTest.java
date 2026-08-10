package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class GeminiQueryServiceTest {

    @Test
    public void testInit() {
        GeminiRequestService geminiRequest = EasyMock.createMock(GeminiRequestService.class);
        GeminiRouter geminiRouter = EasyMock.createMock(GeminiRouter.class);
        EasyMock.replay(geminiRequest, geminiRouter);
        GeminiQueryService service = new GeminiQueryService();
        service.setGeminiRequestService(geminiRequest);
        service.setGeminiRouter(geminiRouter);
        Assert.assertEquals(geminiRequest, service.request());
        Assert.assertEquals(geminiRouter, service.router());
        EasyMock.verify(geminiRequest, geminiRouter);
    }

    @Test
    public void testStream() throws Exception {
        GeminiRequestService gemini = EasyMock.createMock(GeminiRequestService.class);
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        GeminiRouter router = EasyMock.createMock(GeminiRouter.class);
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        NotifierServiceImpl notifier = EasyMock.createMock(NotifierServiceImpl.class);
        HistoryStore history = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(gemini, router, signal, request, history);
        GeminiQueryService service = new GeminiQueryService();
        service.setHistoryStore(history);
        service.setNotifierService(notifier);
        service.setGeminiRequestService(gemini);
        service.setGeminiRouter(router);
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNamesService(new NamesServiceImpl());
        GoogleStream stream = service.stream(signal, request);
        Assert.assertEquals(stream.getHistoryStore(), history);
        Assert.assertEquals(stream.getRequest(), request);
        Assert.assertEquals(stream.getNotifierService(), notifier);
        Assert.assertEquals(stream.getSignalStream(), signal);
        EasyMock.verify(gemini, router, signal, request, history);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = GeminiQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = GeminiQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
    @Test
    public void testGetters() {
        GeminiQueryService service = new GeminiQueryService();
        GeminiRequestService req = new GeminiRequestService();
        GeminiRouter router = new GeminiRouter();
        service.setGeminiRequestService(req);
        service.setGeminiRouter(router);
        Assert.assertEquals(req, service.getGeminiRequestService());
        Assert.assertEquals(router, service.getGeminiRouter());
    }
}
