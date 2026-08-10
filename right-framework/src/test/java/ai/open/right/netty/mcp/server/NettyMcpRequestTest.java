package ai.open.right.netty.mcp.server;

import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.workflow.mcp.server.McpResponse;
import ai.open.right.workflow.notify.Notifier;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.Assert.*;

public class NettyMcpRequestTest {

    private NettyMcpRequest nettyMcpRequest;
    private ChannelHandlerContext mockContext;

    @Before
    public void setUp() {
        // 创建模拟对象
        mockContext = EasyMock.createMock(ChannelHandlerContext.class);

        // 初始化测试对象
        nettyMcpRequest = NettyMcpRequest.builder()
                .context(mockContext)
                .headers(new HashMap<>())
                .content(new HashMap<>())
                .build();
    }

    @Test
    public void testGetSetContext() {
        // 测试getter
        assertSame(mockContext, nettyMcpRequest.getContext());

        // 创建新的模拟对象并测试setter
        ChannelHandlerContext newContext = EasyMock.createMock(ChannelHandlerContext.class);
        nettyMcpRequest.setContext(newContext);
        assertSame(newContext, nettyMcpRequest.getContext());
    }

    @Test
    public void testGetSetHeaders() {
        // 测试初始状态
        assertNotNull(nettyMcpRequest.getHeaders());
        assertTrue(nettyMcpRequest.getHeaders().isEmpty());

        // 设置新的headers并测试
        Map<String, String> newHeaders = new HashMap<>();
        newHeaders.put("testKey", "testValue");
        nettyMcpRequest.setHeaders(newHeaders);

        assertSame(newHeaders, nettyMcpRequest.getHeaders());
        assertEquals("testValue", nettyMcpRequest.getHeaders().get("testKey"));
    }

    @Test
    public void testGetSetContent() {
        // 测试初始状态
        assertNotNull(nettyMcpRequest.getContent());
        assertTrue(nettyMcpRequest.getContent().isEmpty());

        // 设置新的content并测试
        Map<String, Object> newContent = new HashMap<>();
        newContent.put("testKey", "testValue");
        nettyMcpRequest.setContent(newContent);

        assertSame(newContent, nettyMcpRequest.getContent());
        assertEquals("testValue", nettyMcpRequest.getContent().get("testKey"));
    }

    @Test
    public void testGetSetTrace() {
        // 测试初始状态
        assertNull(nettyMcpRequest.getTrace());

        // 设置trace并测试
        nettyMcpRequest.setTrace("testTrace");
        assertEquals("testTrace", nettyMcpRequest.getTrace());

        // 测试setTrace方法的默认值行为
        nettyMcpRequest.setTrace("newTrace");
        assertEquals("testTrace", nettyMcpRequest.getTrace()); // 不应改变已有值
    }

    @Test
    public void testGetMethod() {
        // 测试正常情况
        nettyMcpRequest.getContent().put(NettyMcpRequest.METHOD, "testMethod");
        assertEquals("TESTMETHOD", nettyMcpRequest.getMethod());

        // 测试空方法名的情况（应抛出异常）
        nettyMcpRequest.getContent().remove(NettyMcpRequest.METHOD);
        assertThrows(IllegalArgumentException.class, () -> nettyMcpRequest.getMethod());
    }

    @Test
    public void testGetId() {
        // 测试获取id
        nettyMcpRequest.getContent().put(NettyMcpRequest.ID, "testId");
        assertEquals("testId", nettyMcpRequest.getId());

        // 测试id不存在的情况
        nettyMcpRequest.getContent().remove(NettyMcpRequest.ID);
        assertNull(nettyMcpRequest.getId());
    }

    @Test
    public void testInit() {
        // 测试从headers中获取trace
        nettyMcpRequest.getHeaders().put(NettyMcpRequest.TRACE, "headerTrace");
        nettyMcpRequest.init();
        assertEquals("headerTrace", nettyMcpRequest.getTrace());

        // 测试headers为空的情况
        nettyMcpRequest.setHeaders(null);
        nettyMcpRequest.setTrace(null);
        assertNotNull(nettyMcpRequest.getTrace());
        Assert.assertEquals("headerTrace", nettyMcpRequest.getTrace());
    }

    @org.junit.jupiter.api.Test
    public void testWrite() throws Exception {
        McpResponse response = EasyMock.createMock(McpResponse.class);
        EasyMock.expect(response.getNotifier()).andReturn(false).anyTimes();
        EasyMock.expect(response.getWrap()).andReturn(false).anyTimes();
        EasyMock.expect(response.getResult()).andReturn(new HashMap<>()).anyTimes();
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(ctx.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(ctx.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(ctx.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(ctx, response, attributeCors, returnFuture, closeFuture);
        Map<String, Object> content = new HashMap<>();
        content.put("id", "123");
        NettyMcpRequest request = NettyMcpRequest.builder().context(ctx).content(content).build();
        request.write(response);
    }

    @org.junit.jupiter.api.Test
    public void testError() throws Exception {
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(ctx.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(ctx.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(ctx.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(ctx.close()).andReturn(closeFuture).anyTimes();
        EasyMock.replay(ctx, attributeCors, returnFuture, closeFuture);
        Map<String, Object> content = new HashMap<>();
        content.put("id", "123");
        NettyMcpRequest request = NettyMcpRequest.builder().context(ctx).content(content).build();
        request.error("error message");
    }
}

