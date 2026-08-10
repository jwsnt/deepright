package ai.open.right.netty.chat;

import ai.open.right.netty.NettyInputBuffer;
import io.netty.buffer.ByteBufAllocator;
import org.junit.Assert;
import org.junit.Test;

public class NettyInputBufferTest {

    @Test
    public void testInit() throws Exception {
        NettyInputBuffer nettyInputBuffer = new NettyInputBuffer(ByteBufAllocator.DEFAULT.buffer());
        Assert.assertEquals(nettyInputBuffer.available(), 0);
    }

    @Test
    public void testInitWithSize() throws Exception {
        NettyInputBuffer nettyInputBuffer = new NettyInputBuffer(ByteBufAllocator.DEFAULT.buffer(), 1024);
        Assert.assertEquals(nettyInputBuffer.available(), 0);
    }
}
