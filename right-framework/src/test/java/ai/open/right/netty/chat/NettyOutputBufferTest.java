package ai.open.right.netty.chat;

import ai.open.right.netty.NettyOutputBuffer;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.PooledByteBufAllocator;
import org.junit.Test;

public class NettyOutputBufferTest {

    @Test
    public void testNettyOutputBufferInit1() {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        NettyOutputBuffer buffer = new NettyOutputBuffer(buf, 1024);
    }

    @Test
    public void testNettyOutputBufferInit2() {
        ByteBuf buf = PooledByteBufAllocator.DEFAULT.directBuffer();
        NettyOutputBuffer buffer = new NettyOutputBuffer(buf);
    }
}
