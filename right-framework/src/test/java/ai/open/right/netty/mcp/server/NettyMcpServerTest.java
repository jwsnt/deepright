package ai.open.right.netty.mcp.server;

import ai.open.right.netty.NettyServer;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandler;
import io.netty.channel.ChannelPipeline;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.SocketChannelConfig;
import io.netty.handler.timeout.IdleStateHandler;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class NettyMcpServerTest {

    @Test
    public void test() {
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
        NettyMcpServer server = new NettyMcpServer();
        ChannelHandler[] channelHandlers = new ChannelHandler[5];
        channelHandlers[0] = EasyMock.createMock(IdleStateHandler.class);
        ChannelPipeline pipeline = EasyMock.createMock(ChannelPipeline.class);
        EasyMock.expect(pipeline.addLast(channelHandlers)).andReturn(pipeline).anyTimes();
        EasyMock.expect(channel.pipeline()).andReturn(pipeline).anyTimes();
        EasyMock.replay(channel, channelConfig, pipeline);
        server.setBufferRecv(1024);
        server.setBufferSend(1024);
        server.setIdle(1024);
        Assert.assertEquals(server.new McpInitializer(100, 100).getChannelHandler().length, 5);
        EasyMock.verify(channel, pipeline);
    }

    @Test
    public void testInit() throws Exception {
        NettyMcpHandler mcpHandler = EasyMock.createMock(NettyMcpHandler.class);
        NettyMcpServer.InitConfig initConfig = new NettyMcpServer.InitConfig();
        initConfig.setMcpHandler(mcpHandler);
        initConfig.setBinding("BIND");
        initConfig.setIdle(99);
        initConfig.setPort(10086);
        initConfig.setRequestMax(10085);
        initConfig.setEventLoopChildren(10084);
        initConfig.setMaxChunkSize(10083);
        initConfig.setMaxHeaderSize(10082);
        initConfig.setEventLoopParent(10081);
        initConfig.setBufferRecv(10080);
        initConfig.setBufferSend(10079);
        NettyMcpServer nettyMcpServer = initConfig.nettyMcpServer();
        Assert.assertEquals(mcpHandler, nettyMcpServer.getMcpHandler());
        Assert.assertEquals("BIND", nettyMcpServer.getBinding());
        Assert.assertEquals(Integer.valueOf(10081), nettyMcpServer.getEventLoopParent());
        Assert.assertEquals(Integer.valueOf(10084), nettyMcpServer.getEventLoopChildren());
        Assert.assertEquals(Integer.valueOf(99), nettyMcpServer.getIdle());
        Assert.assertEquals(Integer.valueOf(10083), nettyMcpServer.getMaxChunkSize());
        Assert.assertEquals(Integer.valueOf(10082), nettyMcpServer.getMaxHeaderSize());
        Assert.assertEquals(Integer.valueOf(10086), nettyMcpServer.getPort());
        Assert.assertEquals(Integer.valueOf(10080), nettyMcpServer.getBufferRecv());
        Assert.assertEquals(Integer.valueOf(10079), nettyMcpServer.getBufferSend());
    }

    @Test
    public void testHandler() {
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
        NettyMcpServer server = new NettyMcpServer();
        ChannelHandler[] channelHandlers = new ChannelHandler[5];
        channelHandlers[0] = EasyMock.createMock(IdleStateHandler.class);
        ChannelPipeline pipeline = EasyMock.createMock(ChannelPipeline.class);
        EasyMock.expect(pipeline.addLast(channelHandlers)).andReturn(pipeline).anyTimes();
        EasyMock.expect(channel.pipeline()).andReturn(pipeline).anyTimes();
        EasyMock.replay(channel, channelConfig, pipeline);
        server.setBufferRecv(1024);
        server.setBufferSend(1024);
        server.setIdle(1024);
        NettyServer.ProxyChannelInitializer initializer = server.buildHandler();
        Assert.assertEquals(initializer.getChannelHandler().length, 5);
        EasyMock.verify(channel, pipeline);
    }


    @Test
    public void testHashCode1() throws Exception {
        Object object = NettyMcpServer.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = NettyMcpServer.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
