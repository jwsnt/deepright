package ai.open.right.netty.mcp;

import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.workflow.mcp.server.McpResponse;
import ai.open.right.workflow.mcp.server.cmd.McpCmdResponse;
import com.google.common.collect.ImmutableMap;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class NettyMcpRequestTest {

    @Test
    public void testTrace() {
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .headers(ImmutableMap.of(NettyMcpRequest.TRACE, "OK"))
                .build();
        Assert.assertEquals("OK", nettyMcpRequest.init().getTrace());
    }

    @Test
    public void testEmptyHeader() {
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .build();
        Assert.assertTrue(nettyMcpRequest.getHeaders().isEmpty());
    }

    @Test
    public void testId() {
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .content(ImmutableMap.of(NettyMcpRequest.ID, "OK"))
                .build();
        Assert.assertEquals("OK", nettyMcpRequest.getId());
    }

    @Test
    public void testMethod() {
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .content(ImmutableMap.of(NettyMcpRequest.METHOD, "OK"))
                .build();
        Assert.assertEquals("OK", nettyMcpRequest.getMethod());
    }

    @Test
    public void testWrite() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(io.netty.buffer.ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .content(ImmutableMap.of("id", "WORLD"))
                .context(context)
                .build();
        McpResponse mcpResponse = McpCmdResponse.builder()
                .result(ImmutableMap.of("HELLO", "WROLD"))
                .build();
        nettyMcpRequest.write(mcpResponse);
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testError() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(io.netty.buffer.ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .content(ImmutableMap.of("id", "WORLD"))
                .context(context)
                .build();
        nettyMcpRequest.error("HELLO WROLD");
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
    }

}
