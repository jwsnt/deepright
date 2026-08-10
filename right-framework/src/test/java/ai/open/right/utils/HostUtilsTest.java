package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

public class HostUtilsTest {

    @Test
    public void getHostName_returnsNonNull() throws Exception {
        Assert.assertNotNull(HostUtils.getHostName());
    }

    @Test
    public void getHostName_returnsNonEmpty() throws Exception {
        String name = HostUtils.getHostName();
        Assert.assertNotNull(name);
        Assert.assertFalse(name.isEmpty());
        Assert.assertFalse(name.trim().isEmpty());
    }

    @Test
    public void getHostName_returnsConsistentValue() throws Exception {
        String first = HostUtils.getHostName();
        String second = HostUtils.getHostName();
        Assert.assertNotNull(first);
        Assert.assertEquals(first, second);
    }

    @Test
    public void getHostName_resultReasonableLength() throws Exception {
        String name = HostUtils.getHostName();
        Assert.assertNotNull(name);
        Assert.assertTrue("hostname length should be reasonable", name.length() >= 1 && name.length() <= 253);
    }
}
