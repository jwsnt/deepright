package ai.open.right.netty.a2a.distribute;

import ai.open.right.netty.a2a.distribute.NettyDistributor;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.a2a.server.A2ADistributor;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class NettyDistributorInitConfigTest {

    @Test
    public void shouldCreateNettyDistributorWithProvidedProperties() throws Exception {
        NettyDistributor.InitConfig init = new NettyDistributor.InitConfig();

        A2ADistributor a2aTaskService = EasyMock.createMock(A2ADistributor.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);

        EasyMock.replay(a2aTaskService, traceService);

        // 设置属性
        init.setA2aDistributor(a2aTaskService);
        init.setTraceService(traceService);

        NettyDistributor bean = init.nettyDistributor();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyDistributor);

        EasyMock.verify(a2aTaskService, traceService);
    }

    @Test
    public void shouldCreateNettyDistributorWithDefaults() throws Exception {
        NettyDistributor.InitConfig init = new NettyDistributor.InitConfig();
        A2ADistributor a2aTaskService = EasyMock.createMock(A2ADistributor.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        EasyMock.replay(a2aTaskService, traceService);
        init.setTraceService(traceService);
        init.setA2aDistributor(a2aTaskService);
        NettyDistributor bean = init.nettyDistributor();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyDistributor);
        Assert.assertEquals(traceService, bean.getTraceService());
        Assert.assertEquals(a2aTaskService, bean.getA2aDistributor());
        EasyMock.verify(a2aTaskService, traceService);
    }
}
