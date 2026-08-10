package ai.open.right.netty;

import io.netty.channel.ChannelHandlerContext;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Test;

public class NettyCaptorTest {

    @Test
    public void testAnonymousCaptor() throws Exception {
        final boolean[] called = {false};
        NettyCaptor captor = new NettyCaptor() {
            @Override
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable e) throws Exception {
                called[0] = true;
            }
        };
        captor.exceptionCaught(EasyMock.createMock(ChannelHandlerContext.class), new RuntimeException());
        org.junit.jupiter.api.Assertions.assertTrue(called[0]);
    }
}
