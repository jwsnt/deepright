package ai.open.right.netty.chat.distribute;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.listener.EventListenerService;
import ai.open.right.trace.TraceService;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.Workflow;

public class NettyDistributorInitConfigTest {

    @Test
    public void shouldCreateDistributorWithInjectedMocks() throws Exception {
        NettyDistributor.InitConfig init = new NettyDistributor.InitConfig();

        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        TraceService traceService = EasyMock.createMock(TraceService.class);
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        NettyTrack nettyTrack = EasyMock.createMock(NettyTrack.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);

        init.setEventListenerService(eventListenerService);
        init.setTraceService(traceService);
        init.setTokenMapping(tokenMapping);
        init.setNettyTrack(nettyTrack);
        init.setWorkflow(workflow);

        NettyDistributor bean = init.distributor();

        Assert.assertSame(eventListenerService, bean.getEventListenerService());
        Assert.assertSame(traceService, bean.getTraceService());
        Assert.assertSame(tokenMapping, bean.getTokenMapping());
        Assert.assertSame(nettyTrack, bean.getNettyTrack());
        Assert.assertSame(workflow, bean.getWorkflow());
    }
}
