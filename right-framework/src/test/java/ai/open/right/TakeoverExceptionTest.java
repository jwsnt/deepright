package ai.open.right;

import org.junit.Assert;
import org.junit.Test;

public class TakeoverExceptionTest {

    @Test
    public void testCodeAndInheritance() {
        TakeoverException takeoverException = TakeoverException.SIGNAL;
        Assert.assertEquals(Integer.valueOf(-101), takeoverException.getCode());
        Assert.assertTrue(takeoverException instanceof WorkflowException);
    }

    @Test
    public void needSlient_returnsThis() {
        TakeoverException e = TakeoverException.SIGNAL;
        TakeoverException result = e.needSilent();
        Assert.assertSame("needSlient() 应返回 this", e, result);
    }
}