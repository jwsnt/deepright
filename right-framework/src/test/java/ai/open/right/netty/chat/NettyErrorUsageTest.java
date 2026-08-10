package ai.open.right.netty.chat;

import ai.open.right.netty.chat.server.http.NettyErrorUsage;
import org.junit.Assert;
import org.junit.Test;

public class NettyErrorUsageTest {

    @Test
    public void test() {
        NettyErrorUsage usage = new NettyErrorUsage();
        Assert.assertEquals(Integer.valueOf(0), usage.getCache());
        Assert.assertEquals(Integer.valueOf(0), usage.getTotal());
    }
}
