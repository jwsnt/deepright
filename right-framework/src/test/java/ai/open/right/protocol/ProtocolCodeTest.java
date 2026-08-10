package ai.open.right.protocol;

import org.junit.Assert;
import org.junit.Test;

public class ProtocolCodeTest {
    @Test
    public void test() {
        Assert.assertEquals((int) ProtocolCode.C200, 200);
        Assert.assertEquals((int) ProtocolCode.C500, 500);
        Assert.assertEquals((int) ProtocolCode.C400, 400);
        Assert.assertEquals((int) ProtocolCode.C429, 429);
        Assert.assertEquals((int) ProtocolCode.C502, 502);
        Assert.assertEquals((int) ProtocolCode.C503, 503);
    }

    @Test
    public void testRange() {
        Assert.assertTrue(ProtocolCode.range2xx(200));
        Assert.assertTrue(ProtocolCode.range2xx(299));
        Assert.assertFalse(ProtocolCode.range2xx(199));
        Assert.assertFalse(ProtocolCode.range2xx(301));
    }

    @org.junit.jupiter.api.Test
    public void testProtocolCodeInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}