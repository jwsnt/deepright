package ai.open.right.workflow.a2a;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.a2a.protocol.Task;
import ai.open.right.workflow.flow.llm.Segment;
import org.junit.Assert;
import org.junit.Test;

public class A2ADataTest {

    @Test
    public void test() {
        A2AData a2AData = new A2AData();
        Assert.assertNull(a2AData.get("OK"));
        a2AData.put("OK", "YES");
        Assert.assertEquals("YES", a2AData.get("OK"));
    }

    @Test
    public void testSupport() {
        A2AData a2AData = new A2AData();
        Assert.assertFalse(a2AData.isSupport(Task.PROTOCOL));
        a2AData.put("internal", Task.PROTOCOL);
        Assert.assertTrue(a2AData.isSupport(Task.PROTOCOL));
        a2AData.reset();
        Assert.assertFalse(a2AData.isSupport(Task.PROTOCOL));
    }

    @Test
    public void testSegment() {
        Segment segment = ObjectBuilder.buildSegment();
        A2AData a2AData = new A2AData();
        Assert.assertNull(a2AData.getSegment());
        a2AData.setSegment(segment);
        Assert.assertNotNull(a2AData.getSegment());
        a2AData.bindSegment(segment);
        Assert.assertNotNull(a2AData.getSegment());
    }
}
