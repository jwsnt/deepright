package ai.open.right.netty.a2a.server;

import ai.open.right.netty.a2a.distribute.NettyDistributor;
import org.junit.Assert;
import org.junit.Test;

public class NettyA2AHandlerInitConfigTest {

    @Test
    public void shouldCreateNettyHandlerWithProvidedProperties() throws Exception {
        NettyA2AHandler.InitConfig init = new NettyA2AHandler.InitConfig();

        NettyDistributor distribute = new NettyDistributor();

        // 设置属性
        init.setDistributor(distribute);
        init.setAutoDump("/var/tmp/a2a-harness-test");

        NettyA2AHandler bean = init.nettyA2AHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyA2AHandler);
        Assert.assertEquals("/var/tmp/a2a-harness-test", bean.getAutoDump());
        Assert.assertSame(distribute, bean.getDistributor());
    }

    @Test
    public void shouldCreateNettyHandlerWithDefaults() throws Exception {

        NettyA2AHandler.InitConfig init = new NettyA2AHandler.InitConfig();
        NettyA2AHandler bean = init.nettyA2AHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyA2AHandler);
    }
}
