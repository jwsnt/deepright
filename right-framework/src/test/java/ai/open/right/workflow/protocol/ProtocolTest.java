package ai.open.right.workflow.protocol;

import ai.open.right.protocol.Protocol;
import org.junit.Assert;
import org.junit.Test;

public class ProtocolTest {

    @Test
    public void test() {
        Assert.assertEquals(Protocol.CHAT, "chat");
        Assert.assertEquals(Protocol.TOOL,"tool");
    }
}
