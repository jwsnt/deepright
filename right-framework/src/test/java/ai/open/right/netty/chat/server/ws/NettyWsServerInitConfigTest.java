package ai.open.right.netty.chat.server.ws;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class NettyWsServerInitConfigTest {

    @Test
    public void shouldCreateNettyWsServerWithProvidedProperties() throws Exception {
        NettyWsServer.InitConfig init = new NettyWsServer.InitConfig();

        NettyWsHandler handler = EasyMock.createMock(NettyWsHandler.class);

        EasyMock.replay(handler);

        // 设置属性
        init.setEventLoopChildren(2);
        init.setEventLoopParent(1);
        init.setMaxInitialLineLength(8192);
        init.setMaxHeaderSize(16384);
        init.setMaxChunkSize(16384);
        init.setRequestMax(1024);
        init.setBinding("127.0.0.1");
        init.setIdle(5000);
        init.setPort(8080);
        init.setHandler(handler);

        NettyWsServer bean = init.nettyWsServer();

        // 验证bean被创建
        Assert.assertNotNull(bean);
        Assert.assertEquals(handler, bean.getHandler());
        Assert.assertTrue(bean instanceof NettyWsServer);

        EasyMock.verify(handler);
    }

    @Test
    public void shouldCreateNettyWsServerWithDefaults() throws Exception {
        NettyWsServer.InitConfig init = new NettyWsServer.InitConfig();

        NettyWsHandler handler = EasyMock.createMock(NettyWsHandler.class);
        init.setHandler(handler);

        EasyMock.replay(handler);

        NettyWsServer bean = init.nettyWsServer();

        // 验证bean被创建
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyWsServer);

        EasyMock.verify(handler);
    }
}
