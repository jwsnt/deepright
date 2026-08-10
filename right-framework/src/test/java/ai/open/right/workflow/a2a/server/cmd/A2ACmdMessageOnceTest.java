package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.integration.RightService;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.workflow.a2a.A2ARequest;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class A2ACmdMessageOnceTest {

    @Test
    public void testBuildSyncCallable() throws Exception {
        A2ACmdMessageOnce a2ACmdMessageOnce = new A2ACmdMessageOnce();
        Assert.assertEquals(A2ACmdCallableOnce.class, a2ACmdMessageOnce.buildSyncCallable(null, null).getClass());
    }

    @Test
    public void testSupport1() throws Exception {
        A2ACmdMessageOnce a2ACmdMessageOnce = new A2ACmdMessageOnce();
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .path("PATH/A@B")
                .build();
        Assert.assertFalse(a2ACmdMessageOnce.support(a2ARequest));
    }

    @Test
    public void testSupport2() throws Exception {
        A2ACmdMessageOnce a2ACmdMessageOnce = new A2ACmdMessageOnce();
        A2ARequest a2ARequest = EasyMock.createMock(A2ARequest.class);
        EasyMock.expect(a2ARequest.getMethod()).andReturn(A2ACmdMessageOnce.METHOD).anyTimes();
        EasyMock.replay(a2ARequest);
        Assert.assertTrue(a2ACmdMessageOnce.support(a2ARequest));
        EasyMock.verify(a2ARequest);
    }

    @Test
    public void testInit() throws Exception {
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(rightService);
        A2ACmdMessageOnce.InitConfig initConfig = new A2ACmdMessageOnce.InitConfig();
        initConfig.setTimeout4Llm(10086);
        initConfig.setRightService(rightService);
        A2ACmdMessageOnce a2ACmdMessageOnce = initConfig.a2aCmdMessageOnce();
        Assert.assertEquals(a2ACmdMessageOnce.getTimeout4Llm(), initConfig.getTimeout4Llm());
        Assert.assertEquals(a2ACmdMessageOnce.getRightService(), initConfig.getRightService());
        EasyMock.verify(rightService);
    }
}
