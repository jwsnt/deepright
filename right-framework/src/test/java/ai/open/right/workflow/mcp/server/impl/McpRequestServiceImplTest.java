package ai.open.right.workflow.mcp.server.impl;

import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpResponse;
import ai.open.right.workflow.mcp.server.cmd.McpCmdResponse;
import com.google.common.collect.ImmutableMap;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Test;

public class McpRequestServiceImplTest {

    @Test
    public void testWriter() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .content(ImmutableMap.of("id", "WORLD", NettyMcpRequest.METHOD, McpMethod.KEY_RESOURCES_LIST))
                .context(context)
                .build();
        McpResponse mcpResponse = McpCmdResponse.builder()
                .result(ImmutableMap.of("HELLO", "WROLD"))
                .build();
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
        McpDistributorImpl mcpRequestService = new McpDistributorImpl();
        McpCmdExportService exportService = EasyMock.createMock(McpCmdExportService.class);
        exportService.cmd(nettyMcpRequest);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(exportService);
        mcpRequestService.setCmdServices(ImmutableMap.of(McpMethod.KEY_RESOURCES_LIST, exportService));
        mcpRequestService.distribute(nettyMcpRequest);
        EasyMock.verify(exportService);
    }

    @Test
    public void testError() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context, attributeCors, closeFuture, returnFuture);
        NettyMcpRequest nettyMcpRequest = NettyMcpRequest.builder()
                .content(ImmutableMap.of("id", "WORLD", NettyMcpRequest.METHOD, McpMethod.KEY_RESOURCES_LIST))
                .context(context)
                .build();
        EasyMock.verify(context, attributeCors, closeFuture, returnFuture);
        McpDistributorImpl mcpRequestService = new McpDistributorImpl();
        McpCmdExportService exportService = EasyMock.createMock(McpCmdExportService.class);
        exportService.cmd(nettyMcpRequest);
        EasyMock.expectLastCall().andThrow(new RuntimeException("RUNTIME")).anyTimes();
        EasyMock.replay(exportService);
        mcpRequestService.setCmdServices(ImmutableMap.of(McpMethod.KEY_RESOURCES_LIST, exportService));
        mcpRequestService.distribute(nettyMcpRequest);
        EasyMock.verify(exportService);
    }
}
