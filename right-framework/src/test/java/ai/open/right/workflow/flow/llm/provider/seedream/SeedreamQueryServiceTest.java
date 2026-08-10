package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.impl.RedisHistoryStore;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

/**
 * SeedQueryService 及其内部类 InitConfig 的单元测试
 */
public class SeedreamQueryServiceTest {

    @Test
    public void testStream() throws Exception {
        SeedreamQueryService service = new SeedreamQueryService();
        service.setNamesService(new NamesServiceImpl());
        service.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setHistoryStore(new RedisHistoryStore());
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.hasChain()).andReturn(false).anyTimes();
        EasyMock.expect(request.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(request.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(request.getSuffix()).andReturn("").anyTimes();
        // 确保 Mock 对象进入 replay 状态，修复 NullPointerException
        EasyMock.replay(request, signalStream);

        // 测试 stream 对象的创建
        SeedreamStream stream = service.stream(signalStream, request);
        Assertions.assertNotNull(stream);
    }

    @Test
    public void testRequestAndRouter() throws Exception {
        SeedreamQueryService service = new SeedreamQueryService();
        SeedreamRequestService requestService = EasyMock.createMock(SeedreamRequestService.class);
        SeedreamRouter router = EasyMock.createMock(SeedreamRouter.class);

        service.setSeedRequestService(requestService);
        service.setSeedRouter(router);

        // 验证内部获取 request 和 router 的逻辑
        Assertions.assertEquals(requestService, service.request());
        Assertions.assertEquals(router, service.router());
        Assertions.assertEquals(requestService, service.getSeedRequestService());
        Assertions.assertEquals(router, service.getSeedRouter());
    }

    @Test
    public void testQuery() throws Exception {
        SeedreamQueryService service = new SeedreamQueryService();
        service.setNamesService(new NamesServiceImpl());
        service.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        service.setMediaInlineService(new MediaInlineServiceImpl());
        service.setHistoryStore(new RedisHistoryStore());
        SeedreamRequestService requestService = EasyMock.createMock(SeedreamRequestService.class);
        SeedreamRouter router = EasyMock.createMock(SeedreamRouter.class);

        service.setSeedRequestService(requestService);
        service.setSeedRouter(router);

        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        SeedreamRequest request = EasyMock.createMock(SeedreamRequest.class);
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
        router.route(EasyMock.eq(request), EasyMock.eq(llmConfig), EasyMock.anyObject(SeedreamStream.class));
        EasyMock.expectLastCall().anyTimes();

        EasyMock.replay(requestService, router, request, signalStream);

        // 执行查询逻辑
        service.query(llmQuery, llmConfig, signalStream);

        // 验证交互
        EasyMock.verify(requestService, router);
    }

    @Test
    public void testInitConfig() throws Exception {
        SeedreamQueryService.InitConfig initConfig = new SeedreamQueryService.InitConfig();
        SeedreamRequestService requestService = EasyMock.createMock(SeedreamRequestService.class);
        SeedreamRouter router = EasyMock.createMock(SeedreamRouter.class);

        initConfig.setSeedRequestService(requestService);
        initConfig.setSeedRouter(router);

        // 验证 InitConfig 的属性设置与获取
        Assertions.assertEquals(requestService, initConfig.getSeedRequestService());
        Assertions.assertEquals(router, initConfig.getSeedRouter());

        // 测试 Bean 初始化方法
        SeedreamQueryService result = initConfig.seedreamQueryService();
        Assertions.assertNotNull(result);
    }
}

