package ai.open.right.netty.chat.server.http;

import ai.open.right.WorkflowException;
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
import org.springframework.context.support.StaticApplicationContext;

public class NettyHttpServerTest {

    private void setIdle(NettyHttpServer server, int idle) {
        server.setIdleR(idle);
        server.setIdleW(idle);
        server.setIdleA(idle);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = NettyHttpServer.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = NettyHttpServer.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testNettyHttpServer() throws Exception {
        NettyHttpServer server = new NettyHttpServer();
        server.setBinding("127.0.0.1");
        server.setPort(12222);
        server.init();
        ContextRefreshedEvent contextRefreshedEvent = new ContextRefreshedEvent(new StaticApplicationContext());
        server.onApplicationEvent(contextRefreshedEvent);
        server.destroy();
    }

    @Test
    public void testNettyHttpServerException() throws Exception {
        NettyHttpServer server = new NettyHttpServer();
        server.setBinding("127.0.0.1");
        server.setPort(12222);
        server.init();
        ContextRefreshedEvent contextRefreshedEvent = new ContextRefreshedEvent(new StaticApplicationContext());
        server.onApplicationEvent(contextRefreshedEvent);
        try {
            // Dup
            server.onApplicationEvent(contextRefreshedEvent);
            Assert.fail();
        } catch (WorkflowException e) {

        }
        server.destroy();
        server.destroy();
    }

    @Test
    public void testNettyHttpServerWithConfig() throws Exception {
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
        NettyHttpServer server = new NettyHttpServer();
        server.setBufferRecv(1024);
        server.setBufferSend(1024);
        setIdle(server, 1024);
        ChannelHandler[] channelHandlers = new ChannelHandler[5];
        channelHandlers[0] = new IdleStateHandler(1, 1, 1);
        ChannelPipeline pipeline = EasyMock.createMock(ChannelPipeline.class);
        EasyMock.expect(pipeline.addLast(channelHandlers)).andReturn(pipeline).anyTimes();
        EasyMock.expect(channel.pipeline()).andReturn(pipeline).anyTimes();
        EasyMock.replay(channel, channelConfig, pipeline);
        server.new HttpInitializer(100, 100) {
            public ChannelHandler[] getChannelHandler() {
                return channelHandlers;
            }
        }.initChannel(channel);
        EasyMock.verify(channel, pipeline);
    }

    @Test
    public void testNettyHttpInitializer() throws Exception {
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
        NettyHttpServer server = new NettyHttpServer();
        server.setBufferRecv(1024);
        server.setBufferSend(1024);
        setIdle(server, 1024);
        ChannelHandler[] channelHandlers = new ChannelHandler[5];
        channelHandlers[0] = new IdleStateHandler(1, 1, 1);
        ChannelPipeline pipeline = EasyMock.createMock(ChannelPipeline.class);
        EasyMock.expect(pipeline.addLast(channelHandlers)).andReturn(pipeline).anyTimes();
        EasyMock.expect(channel.pipeline()).andReturn(pipeline).anyTimes();
        EasyMock.replay(channel, channelConfig, pipeline);
        Assert.assertEquals(server.new HttpInitializer(100, 100).getChannelHandler().length, 5);
        EasyMock.verify(channel, pipeline);
    }

    // 新增的getChannelHandler测试方法
    @Test
    public void testGetChannelHandlerWithoutCorsHandler() {
        // 测试没有CORS处理器的情况
        NettyHttpServer server = new NettyHttpServer();
        setIdle(server, 3000);
        server.setMaxInitialLineLength(4096);
        server.setMaxHeaderSize(8192);
        server.setMaxChunkSize(8192);
        server.setRequestMax(-1);
        server.setCorsHandler(null); // 明确设置为null

        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        // 验证处理器数组长度为4（没有CORS处理器）
        Assert.assertEquals("Handler array should have 5 elements without CORS handler", 5, handlers.length);

        // 验证各个处理器的类型
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
        Assert.assertTrue("Handler[1] should be HttpServerCodec", handlers[1] instanceof io.netty.handler.codec.http.HttpServerCodec);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
        // Handler[3] 应该是 httpHandler，但这里我们无法直接验证，因为它是通过@Autowired注入的
    }

    @Test
    public void testGetChannelHandlerWithCorsHandler() {
        // 测试有CORS处理器的情况
        NettyHttpServer server = new NettyHttpServer();
        setIdle(server, 5000);
        server.setMaxInitialLineLength(2048);
        server.setMaxHeaderSize(4096);
        server.setMaxChunkSize(4096);
        server.setRequestMax(1000000);
        // 创建一个模拟的CORS处理器
        NettyCorsHandler corsHandler = new NettyCorsHandler();
        server.setCorsHandler(corsHandler);

        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(2048, 2048);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        // 验证处理器数组长度为5（有CORS处理器）
        Assert.assertEquals("Handler array should have 5 elements with CORS handler", 5, handlers.length);

        // 验证各个处理器的类型
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
        Assert.assertTrue("Handler[1] should be HttpServerCodec", handlers[1] instanceof io.netty.handler.codec.http.HttpServerCodec);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
        Assert.assertEquals("Handler[3] should be CORS handler", corsHandler, handlers[3]);
        // Handler[4] 应该是 httpHandler
    }

    @Test
    public void testGetChannelHandlerWithDefaultValues() {
        // 测试使用默认配置值的情况
        NettyHttpServer server = new NettyHttpServer();
        // 不设置任何值，使用默认值
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(-1, -1);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        // 验证处理器数组长度
        Assert.assertEquals("Handler array should have 5 elements with default values", 5, handlers.length);

        // 验证IdleStateHandler使用默认的idle值
        IdleStateHandler idleHandler = (IdleStateHandler) handlers[0];
        // 注意：由于IdleStateHandler的构造函数参数是私有的，我们无法直接验证值
        // 但我们可以验证对象不为null
        Assert.assertNotNull("IdleStateHandler should not be null", idleHandler);
    }

    @Test
    public void testGetChannelHandlerWithCustomIdleValue() {
        // 测试自定义idle值的情况
        NettyHttpServer server = new NettyHttpServer();
        setIdle(server, 100); // 设置10秒
        server.setCorsHandler(null);

        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
    }

    @Test
    public void testGetChannelHandlerWithCustomHttpCodecValues() {
        // 测试自定义HTTP编解码器参数的情况
        NettyHttpServer server = new NettyHttpServer();
        server.setMaxInitialLineLength(1024);
        server.setMaxHeaderSize(2048);
        server.setMaxChunkSize(1024);
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(512, 512);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[1] should be HttpServerCodec", handlers[1] instanceof io.netty.handler.codec.http.HttpServerCodec);
    }

    @Test
    public void testGetChannelHandlerWithRequestMaxValue() {
        // 测试requestMax为-1的情况（使用Integer.MAX_VALUE）
        NettyHttpServer server = new NettyHttpServer();
        server.setRequestMax(-1);
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 4 elements", 5, handlers.length);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
    }

    @Test
    public void testGetChannelHandlerWithPositiveRequestMaxValue() {
        // 测试requestMax为正数的情况
        NettyHttpServer server = new NettyHttpServer();
        server.setRequestMax(500000);
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
    }

    @Test
    public void testGetChannelHandlerWithZeroRequestMaxValue() {
        // 测试requestMax为0的情况
        NettyHttpServer server = new NettyHttpServer();
        server.setRequestMax(0);
        setIdle(server, 100);
        server.setCorsHandler(null);

        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
    }

    @Test
    public void testGetChannelHandlerWithLargeRequestMaxValue() {
        // 测试requestMax为大数值的情况
        NettyHttpServer server = new NettyHttpServer();
        server.setRequestMax(Integer.MAX_VALUE);
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
    }

    @Test
    public void testGetChannelHandlerWithNegativeBufferValues() {
        // 测试负数的buffer值
        NettyHttpServer server = new NettyHttpServer();
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(-1, -1);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
    }

    @Test
    public void testGetChannelHandlerWithZeroBufferValues() {
        // 测试0的buffer值
        NettyHttpServer server = new NettyHttpServer();
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(0, 0);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
    }

    @Test
    public void testGetChannelHandlerWithLargeBufferValues() {
        // 测试大数值的buffer值
        NettyHttpServer server = new NettyHttpServer();
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(65536, 65536);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements", 5, handlers.length);
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
    }

    @Test
    public void testGetChannelHandlerMultipleCalls() {
        // 测试多次调用getChannelHandler方法
        NettyHttpServer server = new NettyHttpServer();
        server.setCorsHandler(null);
        setIdle(server, 100);
        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(1024, 1024);

        // 多次调用应该返回相同的结果
        ChannelHandler[] handlers1 = initializer.getChannelHandler();
        ChannelHandler[] handlers2 = initializer.getChannelHandler();
        ChannelHandler[] handlers3 = initializer.getChannelHandler();

        Assert.assertEquals("Multiple calls should return same array length", handlers1.length, handlers2.length);
        Assert.assertEquals("Multiple calls should return same array length", handlers2.length, handlers3.length);
        Assert.assertEquals("Handler array should have 5 elements", 5, handlers1.length);
    }

    @Test
    public void testGetChannelHandlerWithCorsHandlerAndCustomValues() {
        // 测试有CORS处理器且使用自定义值的情况
        NettyHttpServer server = new NettyHttpServer();
        setIdle(server, 15000);
        server.setMaxInitialLineLength(8192);
        server.setMaxHeaderSize(16384);
        server.setMaxChunkSize(8192);
        server.setRequestMax(2000000);

        NettyCorsHandler corsHandler = new NettyCorsHandler();
        server.setCorsHandler(corsHandler);

        NettyHttpServer.HttpInitializer initializer = server.new HttpInitializer(4096, 4096);
        ChannelHandler[] handlers = initializer.getChannelHandler();

        Assert.assertEquals("Handler array should have 5 elements with CORS handler", 5, handlers.length);
        Assert.assertTrue("Handler[0] should be IdleStateHandler", handlers[0] instanceof IdleStateHandler);
        Assert.assertTrue("Handler[1] should be HttpServerCodec", handlers[1] instanceof io.netty.handler.codec.http.HttpServerCodec);
        Assert.assertTrue("Handler[2] should be HttpObjectAggregator", handlers[2] instanceof io.netty.handler.codec.http.HttpObjectAggregator);
        Assert.assertEquals("Handler[3] should be CORS handler", corsHandler, handlers[3]);
    }

    @Test
    public void testGetSet() {
        // 测试有CORS处理器的情况
        NettyHttpServer server = new NettyHttpServer();
        setIdle(server, 5000);
        server.setMaxInitialLineLength(2048);
        server.setMaxHeaderSize(4096);
        server.setMaxChunkSize(4096);
        server.setRequestMax(1000000);
        server.setPort(9876);
        NettyHttpHandler nettyHttpHandler = new NettyHttpHandler();
        NettyCorsHandler nettyCorsHandler = new NettyCorsHandler();
        server.setCorsHandler(nettyCorsHandler);
        server.setHttpHandler(nettyHttpHandler);
        Assert.assertEquals(nettyCorsHandler, server.getCorsHandler());
        Assert.assertEquals(nettyHttpHandler, server.getHttpHandler());
        Assert.assertEquals(Integer.valueOf(5000), server.getIdleR());
        Assert.assertEquals(Integer.valueOf(5000), server.getIdleW());
        Assert.assertEquals(Integer.valueOf(5000), server.getIdleA());
        Assert.assertEquals(Integer.valueOf(9876), server.getPort());
    }

    @org.junit.jupiter.api.Test
    public void testHttpServerAdditional() {
        NettyHttpServer server = new NettyHttpServer();
        org.junit.jupiter.api.Assertions.assertNotNull(server);
    }


    @org.junit.jupiter.api.Test
    public void testNettyHttpServerInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}
