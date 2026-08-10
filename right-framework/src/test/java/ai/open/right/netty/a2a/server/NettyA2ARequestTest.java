package ai.open.right.netty.a2a.server;

import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.workflow.a2a.server.cmd.A2ACmdResponse;
import com.google.common.collect.ImmutableMap;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;

import java.net.SocketAddress;
import java.util.HashMap;

import static org.junit.Assert.*;

public class NettyA2ARequestTest {
    private ChannelHandlerContext context;
    private NettyA2ARequest request;

    @Before
    public void setUp() {
        context = EasyMock.createMock(ChannelHandlerContext.class);
        request = NettyA2ARequest.builder()
                .context(context)
                .headers(new HashMap<>())
                .content(new HashMap<>())
                .trace("initial-trace")
                .path("/test/path")
                .connectStream(false)
                .build();
    }

    @Test
    public void testInitialization() {
        assertNotNull(request);
        assertFalse(request.getConnectStream());
        assertEquals("initial-trace", request.getTrace());
        assertEquals("/test/path", request.getPath());
        assertNotNull(request.getHeaders());
        assertNotNull(request.getContent());
        request.setContent(ImmutableMap.of("method", "HELLO"));
        assertEquals("HELLO", request.getMethod());
    }

    @Test
    public void testInit() {
        request.getHeaders().put(NettyA2ARequest.TRACE, "header-trace");
        request.setTrace(null);
        NettyA2ARequest initialized = request.init();
        assertEquals("header-trace", initialized.getTrace());
        assertSame(request, initialized);
    }

    @Test
    public void testSetTrace() {
        request.setTrace("new-trace");
        assertEquals("initial-trace", request.getTrace());
        request.setTrace(null);
        request.setTrace("new-trace");
        assertEquals("initial-trace", request.getTrace());
    }

    @Test
    public void testGetHeaders() {
        request.setHeaders(null);
        assertNotNull(request.getHeaders());
        assertTrue(request.getHeaders().isEmpty());
    }

    @Test
    public void testGetId() {
        request.getContent().put(NettyA2ARequest.ID, "12345");
        assertEquals("12345", request.getId());
        request.getContent().remove(NettyA2ARequest.ID);
        assertNull(request.getId());
    }

    @Test
    public void testGetMethod() {
        assertNull(request.getMethod());
        request.setContent(ImmutableMap.of("method", "HELLO"));
        assertEquals("HELLO", request.getMethod());
    }

    @Test
    public void testConnectStream() {
        assertFalse(request.getConnectStream());
        request.setConnectStream(true);
        assertTrue(request.getConnectStream());
        request.setConnectStream(false);
        assertFalse(request.getConnectStream());
    }

    @Test
    public void writeStream() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        this.request.setContext(context);
        this.request.writeStream(A2ACmdResponse.builder()
                .finished(true)
                .id("OK")
                .build());
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        this.request.setContext(null);
    }

    @Test
    public void writeOnce() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        this.request.setContext(context);
        this.request.writeOnce(A2ACmdResponse.builder()
                .finished(true)
                .id("OK")
                .build());
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        this.request.setContext(null);
        Assert.assertNull(this.request.getContext());
    }

    @Test
    public void close() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.close()).andReturn(closeFuture).once();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).once();
        EasyMock.replay(context, closeFuture, returnFuture);
        this.request.setContext(context);
        this.request.close();
        EasyMock.verify(context, closeFuture, returnFuture);
        this.request.setContext(null);
    }
}
