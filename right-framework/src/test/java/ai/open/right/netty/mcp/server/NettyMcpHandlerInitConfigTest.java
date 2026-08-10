package ai.open.right.netty.mcp.server;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.netty.mcp.distribute.NettyDistributor;

public class NettyMcpHandlerInitConfigTest {

    @Test
    public void shouldCreateNettyHandlerWithProvidedProperties() throws Exception {
        NettyMcpHandler.InitConfig init = new NettyMcpHandler.InitConfig();

        NettyDistributor distribute = EasyMock.createMock(NettyDistributor.class);

        EasyMock.replay(distribute);

        // 设置属性
        init.setDistributor(distribute);

        NettyMcpHandler bean = init.nettyMcpHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyMcpHandler);

        EasyMock.verify(distribute);
    }

    @Test
    public void shouldCreateNettyHandlerWithDefaults() throws Exception {

        NettyMcpHandler.InitConfig init = new NettyMcpHandler.InitConfig();
        NettyMcpHandler bean = init.nettyMcpHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyMcpHandler);
    }
}
