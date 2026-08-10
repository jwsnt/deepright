package ai.open.right.listener.impl;

import ai.open.right.listener.EventImpl;
import org.junit.Assert;
import org.junit.Test;

public class EventImplTest {

    @Test
    public void testInit() {
        EventImpl event = new EventImpl();
        Assert.assertEquals(event, event.init());
    }
}
