package ai.open.right.netty.mcp.distribute;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.trace.TraceService;
import ai.open.right.workflow.mcp.server.McpDistributor;

public class NettyDistributorInitConfigTest {

    @Test
    public void shouldCreateNettyDistributorWithProvidedProperties() throws Exception {
        NettyDistributor.InitConfig init = new NettyDistributor.InitConfig();

        McpDistributor mcpTaskService = EasyMock.createMock(McpDistributor.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);

        EasyMock.replay(mcpTaskService, traceService);

        // 设置属性
        init.setMcpDistributor(mcpTaskService);
        init.setTraceService(traceService);

        NettyDistributor bean = init.nettyDistributor();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyDistributor);

        EasyMock.verify(mcpTaskService, traceService);
    }

    @Test
    public void shouldCreateNettyDistributorWithDefaults() throws Exception {
        NettyDistributor.InitConfig init = new NettyDistributor.InitConfig();
        McpDistributor mcpRequestService = EasyMock.createMock(McpDistributor.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.replay(mcpRequestService, traceService);
        init.setTraceService(traceService);
        init.setMcpDistributor(mcpRequestService);
        NettyDistributor bean = init.nettyDistributor();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyDistributor);
        Assert.assertEquals(traceService, bean.getTraceService());
        Assert.assertEquals(mcpRequestService, bean.getMcpDistributor());
        EasyMock.verify(mcpRequestService, traceService);
    }
}
