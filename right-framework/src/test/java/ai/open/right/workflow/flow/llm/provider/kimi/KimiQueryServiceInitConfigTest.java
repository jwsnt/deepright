package ai.open.right.workflow.flow.llm.provider.kimi;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class KimiQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        KimiRequestService requestService = EasyMock.createMock(KimiRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        KimiRouter router = EasyMock.createMock(KimiRouter.class);
        EasyMock.replay(trackFunCallService, requestService, notifierService, historyStore, router);
        KimiQueryService.InitConfig initConfig = new KimiQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setKimiRequestService(requestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setKimiRouter(router);
        KimiQueryService empty = initConfig.kimiQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(requestService, empty.getKimiRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(router, empty.getKimiRouter());
        EasyMock.verify(trackFunCallService, requestService, notifierService, historyStore, router);
    }
}
