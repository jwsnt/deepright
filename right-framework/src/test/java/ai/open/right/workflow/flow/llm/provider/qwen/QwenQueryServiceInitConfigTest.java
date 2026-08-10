package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class QwenQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        QwenRequestService requestService = EasyMock.createMock(QwenRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        QwenRouter router = EasyMock.createMock(QwenRouter.class);
        EasyMock.replay(trackFunCallService, requestService, notifierService, historyStore, router);
        QwenQueryService.InitConfig initConfig = new QwenQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setQwenRequestService(requestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setQwenRouter(router);
        QwenQueryService empty = initConfig.qwenQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(requestService, empty.getQwenRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(router, empty.getQwenRouter());
        EasyMock.verify(trackFunCallService, requestService, notifierService, historyStore, router);
    }
}
