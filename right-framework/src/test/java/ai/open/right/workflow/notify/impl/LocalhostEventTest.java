package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.Segment;
import org.junit.Assert;
import org.junit.Test;

public class LocalhostEventTest {

    @Test
    public void test() {
        Segment segment = ObjectBuilder.buildSegment();
        LocalhostEvent event = new LocalhostEvent(segment);
        Assert.assertEquals(event.getWorkflow(), segment.getWorkflow());
        Assert.assertEquals(event.getType(), LocalhostEvent.TYPE);
        Assert.assertEquals(segment, event.getData());
        Assert.assertEquals(event, event.init());
    }

    @Test
    public void testGet() {
        Segment segment = ObjectBuilder.buildSegment();
        LocalhostEvent event = new LocalhostEvent(segment);
        Assert.assertEquals("UNKNOWN", event.getDevice());
        Assert.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", event.getDimension());
        Assert.assertEquals(segment.getChat(), event.getChat());
        Assert.assertEquals(segment.getBiz(), event.getBiz());
        Assert.assertTrue(segment.getTimestamp() <= event.getNow());
    }
}
