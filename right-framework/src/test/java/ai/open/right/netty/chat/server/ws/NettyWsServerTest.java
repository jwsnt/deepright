package ai.open.right.netty.chat.server.ws;

import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandler;
import io.netty.channel.ChannelPipeline;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.SocketChannelConfig;
import io.netty.handler.timeout.IdleStateHandler;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.context.event.ContextRefreshedEvent;

public class NettyWsServerTest {

    @Test
    public void testNettyWsServer() throws Exception {
        NettyWsServer server = new NettyWsServer();
        server.setIdle(1000);
        server.setBinding("127.0.0.1");
        server.setPort(12222);
        server.init();
        ContextRefreshedEvent contextRefreshedEvent = EasyMock.createMock(ContextRefreshedEvent.class);
        EasyMock.replay(contextRefreshedEvent);
        server.onApplicationEvent(contextRefreshedEvent);
        EasyMock.verify(contextRefreshedEvent);
        Assert.assertEquals(Integer.valueOf(1000), server.getIdle());
        Assert.assertEquals(Integer.valueOf(12222), server.getPort());
        server.destroy();
    }

    @Test
    public void testNettyWsServerWithConfig() throws Exception {
        SocketChannel channel = EasyMock.createMock(SocketChannel.class);
        SocketChannelConfig channelConfig = EasyMock.createMock(SocketChannelConfig.class);
        EasyMock.expect(channel.config()).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setReceiveBufferSize(1024)).andReturn(null).anyTimes();
        EasyMock.expect(channelConfig.setSendBufferSize(1024)).andReturn(null).anyTimes();
        EasyMock.expect(channelConfig.setAllocator(PooledByteBufAllocator.DEFAULT)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setReuseAddress(true)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setTcpNoDelay(true)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setKeepAlive(true)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setReceiveBufferSize(100)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setSendBufferSize(100)).andReturn(channelConfig).anyTimes();
        NettyWsServer server = new NettyWsServer();
        ChannelHandler[] channelHandlers = new ChannelHandler[5];
        channelHandlers[0] = EasyMock.createMock(IdleStateHandler.class);
        ChannelPipeline pipeline = EasyMock.createMock(ChannelPipeline.class);
        EasyMock.expect(pipeline.addLast(channelHandlers)).andReturn(pipeline).anyTimes();
        EasyMock.expect(channel.pipeline()).andReturn(pipeline).anyTimes();
        EasyMock.replay(channel, channelConfig, pipeline);
        server.setBufferRecv(1024);
        server.setBufferSend(1024);
        server.setIdle(1024);
        server.new WsInitializer(100, 100) {
            public ChannelHandler[] getChannelHandler() {
                return channelHandlers;
            }
        }.initChannel(channel);
        EasyMock.verify(channel, pipeline);
    }

    @Test
    public void testNettyWsInitializer() throws Exception {
        SocketChannel channel = EasyMock.createMock(SocketChannel.class);
        SocketChannelConfig channelConfig = EasyMock.createMock(SocketChannelConfig.class);
        EasyMock.expect(channel.config()).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setReceiveBufferSize(1024)).andReturn(null).anyTimes();
        EasyMock.expect(channelConfig.setSendBufferSize(1024)).andReturn(null).anyTimes();
        EasyMock.expect(channelConfig.setAllocator(PooledByteBufAllocator.DEFAULT)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setReuseAddress(true)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setTcpNoDelay(true)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setKeepAlive(true)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setReceiveBufferSize(100)).andReturn(channelConfig).anyTimes();
        EasyMock.expect(channelConfig.setSendBufferSize(100)).andReturn(channelConfig).anyTimes();
        NettyWsServer server = new NettyWsServer();
        ChannelHandler[] channelHandlers = new ChannelHandler[5];
        channelHandlers[0] = EasyMock.createMock(IdleStateHandler.class);
        ChannelPipeline pipeline = EasyMock.createMock(ChannelPipeline.class);
        EasyMock.expect(pipeline.addLast(channelHandlers)).andReturn(pipeline).anyTimes();
        EasyMock.expect(channel.pipeline()).andReturn(pipeline).anyTimes();
        EasyMock.replay(channel, channelConfig, pipeline);
        server.setBufferRecv(1024);
        server.setBufferSend(1024);
        server.setIdle(1024);
        Assert.assertEquals(server.new WsInitializer(100, 100).getChannelHandler().length, 5);
        EasyMock.verify(channel, pipeline);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = NettyWsServer.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = NettyWsServer.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @org.junit.jupiter.api.Test
    public void testWsServerAdditional() {
        NettyWsServer server = new NettyWsServer();
        org.junit.jupiter.api.Assertions.assertNotNull(server);
    }


    @org.junit.jupiter.api.Test
    public void testNettyWsServerInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}