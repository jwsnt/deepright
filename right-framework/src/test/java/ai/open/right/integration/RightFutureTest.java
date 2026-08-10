package ai.open.right.integration;

import ai.open.right.ObjectBuilder;
import ai.open.right.integration.impl.RightServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.ExecutionException;

public class RightFutureTest {

    @Test(expected = ExecutionException.class)
    public void test() throws Exception {
        RightServiceImpl.RightFuture rightFuture = new RightServiceImpl.RightFuture(ObjectBuilder.buildNotifyWriteBack(), null, 1, 1);
        Thread.sleep(10);
        rightFuture.get();
    }

    @Test
    public void setDone_coversSetter_true() {
        RightServiceImpl.RightFuture rightFuture = new RightServiceImpl.RightFuture(ObjectBuilder.buildNotifyWriteBack(), null, 1, 1);
        Assert.assertFalse(rightFuture.isDone());
        rightFuture.setDone(true);
        Assert.assertTrue(rightFuture.getDone());
        Assert.assertTrue(rightFuture.isDone());
    }

    @Test
    public void setDone_coversSetter_false() {
        RightServiceImpl.RightFuture rightFuture = new RightServiceImpl.RightFuture(ObjectBuilder.buildNotifyWriteBack(), null, 1, 1);
        rightFuture.setDone(true);
        rightFuture.setDone(false);
        Assert.assertFalse(rightFuture.getDone());
        Assert.assertFalse(rightFuture.isDone());
    }
}
