package ai.open.right.workflow.flow.llm.provider.minimax;

import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

/**
 * MiniMaxQueryService 及其内部类 InitConfig 的单元测试
 */
public class MiniMaxQueryServiceTest {

    @Test
    public void testRequestAndRouter() {
        MiniMaxQueryService service = new MiniMaxQueryService();
        MiniMaxRequestService requestService = EasyMock.createMock(MiniMaxRequestService.class);
        MiniMaxRouter router = EasyMock.createMock(MiniMaxRouter.class);

        // 设置 Mock 对象
        service.setMiniMaxRequestService(requestService);
        service.setMiniMaxRouter(router);

        // 验证内部获取 request 和 router 的逻辑
        Assertions.assertEquals(requestService, service.request());
        Assertions.assertEquals(router, service.router());
        Assertions.assertEquals(requestService, service.getMiniMaxRequestService());
        Assertions.assertEquals(router, service.getMiniMaxRouter());
    }

    @Test
    public void testInitConfig() throws Exception {
        MiniMaxQueryService.InitConfig initConfig = new MiniMaxQueryService.InitConfig();
        // 修复 typo: MiniMockRequestService -> MiniMaxRequestService
        MiniMaxRequestService requestService = EasyMock.createMock(MiniMaxRequestService.class);
        MiniMaxRouter router = EasyMock.createMock(MiniMaxRouter.class);

        // 设置 InitConfig 的属性
        initConfig.setMiniMaxRequestService(requestService);
        initConfig.setMiniMaxRouter(router);

        // 验证 InitConfig 的属性设置与获取
        Assertions.assertEquals(requestService, initConfig.getMiniMaxRequestService());
        Assertions.assertEquals(router, initConfig.getMiniMaxRouter());

        // 测试 Bean 初始化方法，验证属性是否通过 BeanUtils 拷贝到 service 中
        MiniMaxQueryService result = initConfig.miniMaxQueryService();
        Assertions.assertNotNull(result);
        Assertions.assertEquals(requestService, result.getMiniMaxRequestService());
        Assertions.assertEquals(router, result.getMiniMaxRouter());
    }
}

