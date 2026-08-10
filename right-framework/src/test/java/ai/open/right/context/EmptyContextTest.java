package ai.open.right.context;

import org.junit.Assert;
import org.junit.Test;

public class EmptyContextTest {

    @Test
    public void test() {
        RedirectContext empty = RedirectContext.EMPTY;
        Assert.assertNull(empty.getOriginal());
        Assert.assertNull(empty.getPrevious());
        Assert.assertNull(empty.getInitial());
        Assert.assertEquals(Integer.valueOf(1), empty.getDeepness());
    }

    @Test
    public void testIsEntry() {
        RedirectContext empty = RedirectContext.EMPTY;
        Assert.assertFalse(empty.isEntry());
    }
}
