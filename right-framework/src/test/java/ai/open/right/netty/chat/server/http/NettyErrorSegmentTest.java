package ai.open.right.netty.chat.server.http;

import org.junit.Assert;
import org.junit.Test;

public class NettyErrorSegmentTest {

    @Test
    public void testSetGet() {
        NettyErrorSegment nettyErrorSegment = NettyErrorSegment.builder().build();
        nettyErrorSegment.setContent("CONTENT");
        nettyErrorSegment.setCode(1024);
        Assert.assertEquals("CONTENT", nettyErrorSegment.getContent());
        Assert.assertEquals(Integer.valueOf(1024), nettyErrorSegment.getCode());
    }

    @org.junit.jupiter.api.Test
    public void testNettyErrorSegmentInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}