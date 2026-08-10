package ai.open.right.workflow;

import ai.open.right.netty.chat.NettyInputProxy;
import ai.open.right.workflow.config.TokenMapping;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import io.netty.channel.ChannelHandlerContext;
import org.easymock.EasyMock;
import org.junit.Test;

public class WorkflowProxyTest {


    @Test
    public void testGetContent() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        buf.writeBytes("{}".getBytes());
        NettyInputProxy workflowProxy = new NettyInputProxy(buf);
        ChannelHandlerContext ctx = EasyMock.createMock(ChannelHandlerContext.class);
        TokenMapping tokenMapping = EasyMock.createMock(TokenMapping.class);
        EasyMock.replay(ctx, tokenMapping);
        workflowProxy.buildRequest(ctx, tokenMapping);
        EasyMock.verify(ctx, tokenMapping);
    }

    @Test
    public void testClose() throws Exception {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        NettyInputProxy workflowProxy = new NettyInputProxy(buf);
        workflowProxy.close();
    }
}
