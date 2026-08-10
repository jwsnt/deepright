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

public class VertexQueryServiceTest {


    @Test
    public void testHashCode1() throws Exception {
        Object object = VertexQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = VertexQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testInit() {
        VertexRequestService vertexRequest = EasyMock.createMock(VertexRequestService.class);
        VertexRouter vertexRouter = EasyMock.createMock(VertexRouter.class);
        EasyMock.replay(vertexRequest, vertexRouter);
        VertexQueryService service = new VertexQueryService();
        service.setVertexRequestService(vertexRequest);
        service.setVertexRouter(vertexRouter);
        Assert.assertEquals(vertexRequest, service.request());
        Assert.assertEquals(vertexRouter, service.router());
        EasyMock.verify(vertexRequest, vertexRouter);
    }

    @Test
    public void testStream() throws Exception {
        VertexRequestService gemini = EasyMock.createMock(VertexRequestService.class);
        GoogleRequest request = EasyMock.createMock(GoogleRequest.class);
        VertexRouter router = EasyMock.createMock(VertexRouter.class);
        SignalStream signal = EasyMock.createMock(SignalStream.class);
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        NotifierServiceImpl notifier = EasyMock.createMock(NotifierServiceImpl.class);
        HistoryStore history = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(gemini, router, signal, request, history);
        VertexQueryService service = new VertexQueryService();
        service.setHistoryStore(history);
        service.setNotifierService(notifier);
        service.setVertexRequestService(gemini);
        service.setVertexRouter(router);
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
    public void testGetters() {
        VertexQueryService service = new VertexQueryService();
        VertexRequestService req = new VertexRequestService();
        VertexRouter router = new VertexRouter();
        service.setVertexRequestService(req);
        service.setVertexRouter(router);
        Assert.assertEquals(req, service.getVertexRequestService());
        Assert.assertEquals(router, service.getVertexRouter());
    }
}
