package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.integration.RightService;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2ARequest;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.net.SocketAddress;
import java.util.Map;

public class A2ACmdMessageStreamTest {

    @Test
    public void testBuildSyncCallable() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        A2ACmdMessageStream a2ACmdMessageStream = new A2ACmdMessageStream();
        Assert.assertEquals(A2ACmdCallableStream.class, a2ACmdMessageStream.buildSyncCallable(a2ARequest, null).getClass());
    }

    @Test
    public void testSupport1() throws Exception {
        A2ACmdMessageStream a2ACmdMessageStream = new A2ACmdMessageStream();
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .path("PATH/A@B")
                .build();
        Assert.assertFalse(a2ACmdMessageStream.support(a2ARequest));
    }

    @Test
    public void testSupport2() throws Exception {
        A2ACmdMessageStream a2ACmdMessageStream = new A2ACmdMessageStream();
        A2ARequest a2ARequest = EasyMock.createMock(A2ARequest.class);
        EasyMock.expect(a2ARequest.getMethod()).andReturn(A2ACmdMessageStream.METHOD).anyTimes();
        EasyMock.replay(a2ARequest);
        Assert.assertTrue(a2ACmdMessageStream.support(a2ARequest));
        EasyMock.verify(a2ARequest);
    }

    @Test
    public void testInit() throws Exception {
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.replay(rightService);
        A2ACmdMessageStream.InitConfig initConfig = new A2ACmdMessageStream.InitConfig();
        initConfig.setTimeout4Llm(10086);
        initConfig.setRightService(rightService);
        A2ACmdMessageStream a2ACmdMessageStream = initConfig.a2aCmdMessageStream();
        Assert.assertEquals(a2ACmdMessageStream.getTimeout4Llm(), initConfig.getTimeout4Llm());
        Assert.assertEquals(a2ACmdMessageStream.getRightService(), initConfig.getRightService());
        EasyMock.verify(rightService);
    }
}
