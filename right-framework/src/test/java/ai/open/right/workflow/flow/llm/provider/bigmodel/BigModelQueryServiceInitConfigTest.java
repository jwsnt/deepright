package ai.open.right.workflow.flow.llm.provider.bigmodel;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class BigModelQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        BigModelRequestService cozeRequestService = EasyMock.createMock(BigModelRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        BigModelRouter deepSeekRouter = EasyMock.createMock(BigModelRouter.class);
        EasyMock.replay(trackFunCallService, cozeRequestService, notifierService, historyStore, deepSeekRouter);
        BigModelQueryService.InitConfig initConfig = new BigModelQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setBigModelRequestService(cozeRequestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setBigModelRouter(deepSeekRouter);
        BigModelQueryService empty = initConfig.bigModelQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(cozeRequestService, empty.getBigModelRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(deepSeekRouter, empty.getBigModelRouter());
        EasyMock.verify(trackFunCallService, cozeRequestService, notifierService, historyStore, deepSeekRouter);
    }
}
