package ai.open.right.workflow.protocol;

import ai.open.right.protocol.ProtocolCode;
import org.junit.Assert;
import org.junit.Test;

public class ProtocolCodeTest {

    @Test
    public void test() {
        Assert.assertTrue(ProtocolCode.C200 == 200);
        Assert.assertTrue(ProtocolCode.C500 == 500);
        Assert.assertTrue(ProtocolCode.C400 == 400);
        Assert.assertTrue(ProtocolCode.C401 == 401);
        Assert.assertTrue(ProtocolCode.C502 == 502);
    }
}
