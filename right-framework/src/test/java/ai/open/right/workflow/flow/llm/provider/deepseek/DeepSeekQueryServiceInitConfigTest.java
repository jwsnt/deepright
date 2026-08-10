package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class DeepSeekQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        DeepSeekRequestService cozeRequestService = EasyMock.createMock(DeepSeekRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        DeepSeekRouter deepSeekRouter = EasyMock.createMock(DeepSeekRouter.class);
        EasyMock.replay(trackFunCallService, cozeRequestService, notifierService, historyStore, deepSeekRouter);
        DeepSeekQueryService.InitConfig initConfig = new DeepSeekQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setDeepSeekRequestService(cozeRequestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setDeepSeekRouter(deepSeekRouter);
        DeepSeekQueryService empty = initConfig.deepSeekQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(cozeRequestService, empty.getDeepSeekRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(deepSeekRouter, empty.getDeepSeekRouter());
        EasyMock.verify(trackFunCallService, cozeRequestService, notifierService, historyStore, deepSeekRouter);
    }
}
