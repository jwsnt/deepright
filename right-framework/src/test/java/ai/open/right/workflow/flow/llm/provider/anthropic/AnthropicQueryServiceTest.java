package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.impl.RedisHistoryStore;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import org.checkerframework.checker.units.qual.N;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

/**
 * AnthropicQueryService 及其内部类 InitConfig 的单元测试
 */
public class AnthropicQueryServiceTest {

    @Test
    public void testStream() throws Exception {
        AnthropicQueryService service = new AnthropicQueryService();
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        service.setHistoryStore(new RedisHistoryStore());
        service.setNamesService(new NamesServiceImpl());
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        AnthropicRequest request = EasyMock.createMock(AnthropicRequest.class);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        // 确保 Mock 对象进入 replay 状态，修复 NullPointerException
        EasyMock.replay(request, signalStream);

        // 测试 stream 对象的创建
        AnthropicStream stream = service.stream(signalStream, request);
        Assertions.assertNotNull(stream);
    }

    @Test
    public void testRequestAndRouter() throws Exception {
        AnthropicQueryService service = new AnthropicQueryService();
        AnthropicRequestService requestService = EasyMock.createMock(AnthropicRequestService.class);
        AnthropicRouter router = EasyMock.createMock(AnthropicRouter.class);

        service.setAnthropicRequestService(requestService);
        service.setAnthropicRouter(router);

        // 验证内部获取 request 和 router 的逻辑
        Assertions.assertEquals(requestService, service.request());
        Assertions.assertEquals(router, service.router());
        Assertions.assertEquals(requestService, service.getAnthropicRequestService());
        Assertions.assertEquals(router, service.getAnthropicRouter());
    }

    @Test
    public void testQuery() throws Exception {
        AnthropicQueryService service = new AnthropicQueryService();
        AnthropicRequestService requestService = EasyMock.createMock(AnthropicRequestService.class);
        AnthropicRouter router = EasyMock.createMock(AnthropicRouter.class);
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        service.setHistoryStore(new RedisHistoryStore());
        service.setNamesService(new NamesServiceImpl());
        service.setAnthropicRequestService(requestService);
        service.setAnthropicRouter(router);

        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        AnthropicRequest request = EasyMock.createMock(AnthropicRequest.class);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);

        // 模拟请求配置过程
        EasyMock.expect(requestService.config(llmConfig, llmQuery)).andReturn(request).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        // 模拟路由过程，验证是否调用了 route 方法
        router.route(EasyMock.eq(request), EasyMock.eq(llmConfig), EasyMock.anyObject(AnthropicStream.class));
        EasyMock.expectLastCall().anyTimes();

        EasyMock.replay(requestService, router, request, signalStream);

        // 执行查询逻辑
        service.query(llmQuery, llmConfig, signalStream);

        // 验证交互
        EasyMock.verify(requestService, router);
    }

    @Test
    public void testInitConfig() throws Exception {
        AnthropicQueryService.InitConfig initConfig = new AnthropicQueryService.InitConfig();
        AnthropicRequestService requestService = EasyMock.createMock(AnthropicRequestService.class);
        AnthropicRouter router = EasyMock.createMock(AnthropicRouter.class);

        initConfig.setAnthropicRequestService(requestService);
        initConfig.setAnthropicRouter(router);

        // 验证 InitConfig 的属性设置与获取
        Assertions.assertEquals(requestService, initConfig.getAnthropicRequestService());
        Assertions.assertEquals(router, initConfig.getAnthropicRouter());

        // 测试 Bean 初始化方法
        AnthropicQueryService result = initConfig.anthropicQueryService();
        Assertions.assertNotNull(result);
    }
}

