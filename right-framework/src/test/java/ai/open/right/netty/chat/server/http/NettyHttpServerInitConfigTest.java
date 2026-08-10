package ai.open.right.netty.chat.server.http;

import org.junit.Assert;
import org.junit.Test;

public class NettyHttpServerInitConfigTest {

    private void setIdle(NettyHttpServer.InitConfig init, int idle) {
        init.setIdleR(idle);
        init.setIdleW(idle);
        init.setIdleA(idle);
    }

    @Test
    public void shouldCreateNettyHttpServerWithProvidedProperties() throws Exception {
        NettyHttpServer.InitConfig init = new NettyHttpServer.InitConfig();

        NettyHttpHandler httpHandler = new NettyHttpHandler();
        NettyCorsHandler corsHandler = new NettyCorsHandler();

        // 设置属性
        init.setEventLoopChildren(2);
        init.setEventLoopParent(1);
        init.setMaxInitialLineLength(8192);
        init.setMaxHeaderSize(16384);
        init.setMaxChunkSize(16384);
        init.setRequestMax(1024);
        init.setBinding("127.0.0.1");
        setIdle(init, 5000);
        init.setPort(8080);
        init.setHttpHandler(httpHandler);
        init.setCorsHandler(corsHandler);

        NettyHttpServer bean = init.nettyHttpServer();

        // 验证bean被创建
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyHttpServer);
        Assert.assertEquals(Integer.valueOf(2), bean.getEventLoopChildren());
        Assert.assertEquals(Integer.valueOf(1), bean.getEventLoopParent());
        Assert.assertEquals(Integer.valueOf(8192), bean.getMaxInitialLineLength());
        Assert.assertEquals(Integer.valueOf(16384), bean.getMaxHeaderSize());
        Assert.assertEquals(Integer.valueOf(16384), bean.getMaxChunkSize());
        Assert.assertEquals(Integer.valueOf(1024), bean.getRequestMax());
        Assert.assertEquals("127.0.0.1", bean.getBinding());
        Assert.assertEquals(Integer.valueOf(5000), bean.getIdleR());
        Assert.assertEquals(Integer.valueOf(5000), bean.getIdleW());
        Assert.assertEquals(Integer.valueOf(5000), bean.getIdleA());
        Assert.assertEquals(Integer.valueOf(8080), bean.getPort());
        Assert.assertSame(httpHandler, bean.getHttpHandler());
        Assert.assertSame(corsHandler, bean.getCorsHandler());
    }

    @Test
    public void shouldCreateNettyHttpServerWithDefaults() throws Exception {
        NettyHttpServer.InitConfig init = new NettyHttpServer.InitConfig();

        NettyHttpHandler httpHandler = new NettyHttpHandler();
        init.setHttpHandler(httpHandler);

        NettyHttpServer bean = init.nettyHttpServer();

        // 验证bean被创建
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyHttpServer);
        Assert.assertSame(httpHandler, bean.getHttpHandler());
    }
}
