package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class VertexQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        VertexRequestService requestService = EasyMock.createMock(VertexRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        VertexRouter router = EasyMock.createMock(VertexRouter.class);
        EasyMock.replay(trackFunCallService, requestService, notifierService, historyStore, router);
        VertexQueryService.InitConfig initConfig = new VertexQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setVertexRequestService(requestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setVertexRouter(router);
        VertexQueryService empty = initConfig.vertexQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(requestService, empty.getVertexRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(router, empty.getVertexRouter());
        EasyMock.verify(trackFunCallService, requestService, notifierService, historyStore, router);
    }
}
