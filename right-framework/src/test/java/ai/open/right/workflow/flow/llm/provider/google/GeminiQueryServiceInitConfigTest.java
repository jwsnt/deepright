package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class GeminiQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        GeminiRequestService requestService = EasyMock.createMock(GeminiRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        GeminiRouter router = EasyMock.createMock(GeminiRouter.class);
        EasyMock.replay(trackFunCallService, requestService, notifierService, historyStore, router);
        GeminiQueryService.InitConfig initConfig = new GeminiQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setGeminiRequestService(requestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setGeminiRouter(router);
        GeminiQueryService empty = initConfig.geminiQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(requestService, empty.getGeminiRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(router, empty.getGeminiRouter());
        EasyMock.verify(trackFunCallService, requestService, notifierService, historyStore, router);
    }
}
