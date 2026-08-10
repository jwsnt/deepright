package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.ObjectBuilder;
import ai.open.right.config.HttpClientConfig;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.impl.RedisHistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class OpenAiQueryServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = OpenAiQueryService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = OpenAiQueryService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() {
        TokenStatistic tokenStatistic = EasyMock.createMock(TokenStatistic.class);
        OpenAiRequestService openAiRequestService = new OpenAiRequestService();
        OpenAiRouter openAiRouter = new OpenAiRouter();
        OpenAiQueryService openAiQueryService = new OpenAiQueryService();
        openAiQueryService.setOpenAiRequestService(openAiRequestService);
        openAiQueryService.setTokenStatistic(tokenStatistic);
        openAiQueryService.setOpenAiRouter(openAiRouter);
        Assert.assertEquals(openAiRequestService, openAiQueryService.request());
        Assert.assertEquals(openAiRouter, openAiQueryService.router());
        Assert.assertEquals(tokenStatistic, openAiQueryService.getTokenStatistic());
    }

    @Test
    public void testStream() throws Exception {
        TokenStatistic tokenStatistic = EasyMock.createMock(TokenStatistic.class);
        OpenAiRequestService openAiRequestService = new OpenAiRequestService();
        openAiRequestService.setNamesService(new NamesServiceImpl());
        openAiRequestService.setHistoryStore(new RedisHistoryStore());
        OpenAiRouter router = new OpenAiRouter();
        HttpClientConfig httpClientConfig = new HttpClientConfig();
        httpClientConfig.setSocket4once(1);
        httpClientConfig.setSocket4stream(2);
        httpClientConfig.setConnect4once(3);
        httpClientConfig.setConnect4stream(4);
        httpClientConfig.setRequest4once(5);
        httpClientConfig.setRequest4stream(6);
        router.setHttpClientConfig(httpClientConfig);
        Assert.assertNotNull(router.getHttpClientConfig());
        OpenAiQueryService openAiQueryService = new OpenAiQueryService();
        openAiQueryService.setMediaInlineService(new MediaInlineServiceImpl());
        openAiQueryService.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        openAiQueryService.setOpenAiRequestService(openAiRequestService);
        openAiQueryService.setTokenStatistic(tokenStatistic);
        openAiQueryService.setHistoryStore(new RedisHistoryStore());
        openAiQueryService.setNamesService(new NamesServiceImpl());
        openAiQueryService.setOpenAiRouter(router);
        OpenAiRequest request = EasyMock.createMock(OpenAiRequest.class);
        EasyMock.expect(request.getFunCallStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getToken()).andReturn("Token").anyTimes();
        EasyMock.expect(request.getStream()).andReturn(true).anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false  ).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.replay(request);
        OpenAiStream openAiStream = openAiQueryService.stream(new SignalStream() {
            @Override
            public void signal(SignalExecutor signalExecutor, Message message) throws Exception {

            }

            @Override
            public void finish(Message message) throws Exception {

            }
        }, request);
        Assert.assertNotNull(openAiStream);
    }
}
