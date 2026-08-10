package ai.open.right.workflow.flow.llm.provider.volcengine;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.provider.ProviderStorePolicy;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStream;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStreamFunCall;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class VolcengineQueryServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = VolcengineQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = VolcengineQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() {
        VolcengineRequestService volcengineRequestService = new VolcengineRequestService();
        VolcengineRouter volcengineRouter = new VolcengineRouter();
        VolcengineQueryService volcengineQueryService = new VolcengineQueryService();
        volcengineQueryService.setVolcengineRequestService(volcengineRequestService);
        volcengineQueryService.setVolcengineRouter(volcengineRouter);
        Assert.assertEquals(volcengineRequestService, volcengineQueryService.request());
        Assert.assertEquals(volcengineRouter, volcengineQueryService.router());
    }
    @Test
    public void testGetters() {
        VolcengineQueryService service = new VolcengineQueryService();
        VolcengineRequestService req = new VolcengineRequestService();
        VolcengineRouter router = new VolcengineRouter();
        service.setVolcengineRequestService(req);
        service.setVolcengineRouter(router);
        Assert.assertEquals(req, service.getVolcengineRequestService());
        Assert.assertEquals(router, service.getVolcengineRouter());
    }

    @Test
    public void testStream_buildsFunCallStreamAndForwardsAllDependencies() throws Exception {
        ProviderStorePolicy providerStorePolicy = EasyMock.createMock(ProviderStorePolicy.class);
        TrackFunCallService trackFunCallService = EasyMock.createMock(TrackFunCallService.class);
        MediaInlineService mediaInlineService = EasyMock.createMock(MediaInlineService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        ProviderReason providerReason = EasyMock.createMock(ProviderReason.class);
        TokenStatistic tokenStatistic = EasyMock.createMock(TokenStatistic.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        EasyMock.replay(providerStorePolicy, trackFunCallService, mediaInlineService, notifierService, providerReason, tokenStatistic, historyStore, namesService, signalStream);

        VolcengineQueryService service = requiredService(notifierService, historyStore, namesService);
        service.setProviderStorePolicy(providerStorePolicy);
        service.setTrackFunCallService(trackFunCallService);
        service.setMediaInlineService(mediaInlineService);
        service.setProviderReason(providerReason);
        service.setTokenStatistic(tokenStatistic);
        OpenAiRequest request = request();
        request.setFunCallStream(false);

        OpenAiStream stream = service.stream(signalStream, request);

        Assert.assertTrue(stream instanceof OpenAiStreamFunCall);
        Assert.assertSame(providerStorePolicy, stream.getProviderStorePolicy());
        Assert.assertSame(trackFunCallService, stream.getTrackFunCallService());
        Assert.assertSame(mediaInlineService, stream.getMediaInlineService());
        Assert.assertSame(notifierService, stream.getNotifierService());
        Assert.assertSame(providerReason, stream.getProviderReason());
        Assert.assertSame(tokenStatistic, stream.getTokenStatistic());
        Assert.assertSame(historyStore, stream.getHistoryStore());
        Assert.assertSame(namesService, stream.getNamesService());
        Assert.assertSame(signalStream, stream.getSignalStream());
        Assert.assertSame(request, stream.getRequest());
        EasyMock.verify(providerStorePolicy, trackFunCallService, mediaInlineService, notifierService, providerReason, tokenStatistic, historyStore, namesService, signalStream);
    }

    @Test
    public void testStream_allowsOptionalDependenciesAndSignalStreamToBeNull() throws Exception {
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(notifierService, historyStore, namesService);
        VolcengineQueryService service = requiredService(notifierService, historyStore, namesService);

        OpenAiStream stream = service.stream(null, request());

        Assert.assertTrue(stream instanceof OpenAiStreamFunCall);
        Assert.assertNull(stream.getProviderStorePolicy());
        Assert.assertNull(stream.getTrackFunCallService());
        Assert.assertNull(stream.getMediaInlineService());
        Assert.assertNull(stream.getProviderReason());
        Assert.assertNull(stream.getTokenStatistic());
        Assert.assertNull(stream.getSignalStream());
        EasyMock.verify(notifierService, historyStore, namesService);
    }

    @Test
    public void testStream_rejectsNullNotifierService() throws Exception {
        VolcengineQueryService service = requiredService(null, EasyMock.createMock(HistoryStore.class), EasyMock.createMock(NamesService.class));

        assertInvalidStream(service, request(), "notifier service");
    }

    @Test
    public void testStream_rejectsNullHistoryStore() throws Exception {
        VolcengineQueryService service = requiredService(EasyMock.createMock(NotifierService.class), null, EasyMock.createMock(NamesService.class));

        assertInvalidStream(service, request(), "history store");
    }

    @Test
    public void testStream_rejectsNullNamesService() throws Exception {
        VolcengineQueryService service = requiredService(EasyMock.createMock(NotifierService.class), EasyMock.createMock(HistoryStore.class), null);

        assertInvalidStream(service, request(), "names service");
    }

    @Test
    public void testStream_rejectsNullRequest() throws Exception {
        VolcengineQueryService service = requiredService(EasyMock.createMock(NotifierService.class), EasyMock.createMock(HistoryStore.class), EasyMock.createMock(NamesService.class));

        assertInvalidStream(service, null, "request");
    }

    private static VolcengineQueryService requiredService(NotifierService notifierService, HistoryStore historyStore, NamesService namesService) {
        VolcengineQueryService service = new VolcengineQueryService();
        service.setNotifierService(notifierService);
        service.setHistoryStore(historyStore);
        service.setNamesService(namesService);
        return service;
    }

    private static OpenAiRequest request() {
        OpenAiRequest request = new OpenAiRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        request.setStream(true);
        return request;
    }

    private static void assertInvalidStream(VolcengineQueryService service, OpenAiRequest request, String expectedMessage) throws Exception {
        try {
            service.stream(null, request);
            Assert.fail("Expected stream configuration validation to fail");
        } catch (IllegalArgumentException e) {
            Assert.assertTrue(e.getMessage().toLowerCase().contains(expectedMessage));
        }
    }
}
