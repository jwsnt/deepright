package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class CozeQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        CozeRequestService cozeRequestService = EasyMock.createMock(CozeRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        CozeRouter cozeRouter = EasyMock.createMock(CozeRouter.class);
        EasyMock.replay(trackFunCallService, cozeRequestService, notifierService, historyStore, cozeRouter);
        CozeQueryService.InitConfig initConfig = new CozeQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setCozeRequestService(cozeRequestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setCozeRouter(cozeRouter);
        CozeQueryService empty = initConfig.cozeQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(cozeRequestService, empty.getCozeRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(cozeRouter, empty.getCozeRouter());
        EasyMock.verify(trackFunCallService, cozeRequestService, notifierService, historyStore, cozeRouter);
    }
}
