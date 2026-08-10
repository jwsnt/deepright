package ai.open.right.workflow.flow.llm.provider.coze;

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

public class CozeQueryServiceTest {

    @Test
    public void testInit() {
        CozeRequestService cozeRequsetService = EasyMock.createMock(CozeRequestService.class);
        EasyMock.replay(cozeRequsetService);
        CozeRouter cozeRouter = EasyMock.createMock(CozeRouter.class);
        CozeQueryService service = new CozeQueryService();
        service.setCozeRequestService(cozeRequsetService);
        service.setCozeRouter(cozeRouter);
        Assert.assertEquals(cozeRequsetService, service.request());
        Assert.assertEquals(cozeRouter, service.router());
        EasyMock.verify(cozeRequsetService);
    }

    @Test
    public void testStream() throws Exception {
        CozeRequestService cozeRequsetService = EasyMock.createMock(CozeRequestService.class);
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        CozeRequest cozeRequest = EasyMock.createMock(CozeRequest.class);
        CozeRouter cozeRouter = EasyMock.createMock(CozeRouter.class);
        EasyMock.expect(cozeRequest.getPrefix()).andReturn(null).anyTimes();
        EasyMock.expect(cozeRequest.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(cozeRequest.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(cozeRequest.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        HistoryStore history = EasyMock.createMock(HistoryStore.class);
        EasyMock.replay(cozeRouter, signalStream, cozeRequest, notifierManager, history);
        CozeQueryService service = new CozeQueryService();
        service.setHistoryStore(history);
        service.setNotifierService(notifierManager);
        service.setCozeRequestService(cozeRequsetService);
        service.setCozeRouter(cozeRouter);
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNamesService(new NamesServiceImpl());
        CozeStream stream = service.stream(signalStream, cozeRequest);
        Assert.assertEquals(stream.getHistoryStore(), history);
        Assert.assertEquals(stream.getRequest(), cozeRequest);
        Assert.assertEquals(stream.getNotifierService(), notifierManager);
        Assert.assertEquals(stream.getSignalStream(), signalStream);
        EasyMock.verify(cozeRouter, signalStream, cozeRequest, notifierManager, history);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = CozeQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = CozeQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
