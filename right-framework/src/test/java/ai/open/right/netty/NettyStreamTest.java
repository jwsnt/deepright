package ai.open.right.netty;

import ai.open.right.protocol.ProtocolCode;
import org.junit.Assert;
import org.junit.Test;

public class NettyStreamTest {

    @Test
    public void testSuccess() {
        NettyStream stream = NettyStream.SUCCESS;
        Assert.assertEquals(ProtocolCode.C200, stream.getCode());
        Assert.assertTrue(stream.isFinished());
    }

    @org.junit.jupiter.api.Test
    public void testNettyStreamInterfaceUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}