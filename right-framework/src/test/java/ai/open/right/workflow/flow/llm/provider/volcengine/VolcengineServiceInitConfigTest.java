package ai.open.right.workflow.flow.llm.provider.volcengine;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class VolcengineServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        VolcengineRequestService requestService = EasyMock.createMock(VolcengineRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        VolcengineRouter router = EasyMock.createMock(VolcengineRouter.class);
        EasyMock.replay(trackFunCallService, requestService, notifierService, historyStore, router);
        VolcengineQueryService.InitConfig initConfig = new VolcengineQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setVolcengineRequestService(requestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setVolcengineRouter(router);
        VolcengineQueryService empty = initConfig.volcengineQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(requestService, empty.getVolcengineRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(router, empty.getVolcengineRouter());
        EasyMock.verify(trackFunCallService, requestService, notifierService, historyStore, router);
    }
}
