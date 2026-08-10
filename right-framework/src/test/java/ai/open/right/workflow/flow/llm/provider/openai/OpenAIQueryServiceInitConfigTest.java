package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class OpenAIQueryServiceInitConfigTest {

    @Test
    public void testInit() throws Exception {
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        OpenAiRequestService requestService = EasyMock.createMock(OpenAiRequestService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        OpenAiRouter router = EasyMock.createMock(OpenAiRouter.class);
        EasyMock.replay(trackFunCallService, requestService, notifierService, historyStore, router);
        OpenAiQueryService.InitConfig initConfig = new OpenAiQueryService.InitConfig();
        initConfig.setTrackFunCallService(trackFunCallService);
        initConfig.setOpenAiRequestService(requestService);
        initConfig.setNotifierService(notifierService);
        initConfig.setHistoryStore(historyStore);
        initConfig.setOpenAiRouter(router);
        OpenAiQueryService empty = initConfig.openAiQueryService();
        Assert.assertEquals(trackFunCallService, empty.getTrackFunCallService());
        Assert.assertEquals(requestService, empty.getOpenAiRequestService());
        Assert.assertEquals(notifierService, empty.getNotifierService());
        Assert.assertEquals(historyStore, empty.getHistoryStore());
        Assert.assertEquals(router, empty.getOpenAiRouter());
        EasyMock.verify(trackFunCallService, requestService, notifierService, historyStore, router);
    }
}
