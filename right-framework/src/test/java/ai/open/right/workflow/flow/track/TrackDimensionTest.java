package ai.open.right.workflow.flow.track;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.Dimension;
import org.junit.Assert;
import org.junit.Test;

public class TrackDimensionTest {

    @Test
    public void test() {
        TrackDimension trackDimension = new TrackDimension();
        trackDimension.setWorkflow("W");
        trackDimension.setTrack("T");
        trackDimension.setBiz("B");
        trackDimension.setChat("C");
        trackDimension.setDevice("D");
        Assert.assertEquals(trackDimension.getWorkflow(), "W");
        Assert.assertEquals("T", trackDimension.getTrack());
        Assert.assertEquals("B", trackDimension.getBiz());
        Assert.assertEquals("C", trackDimension.getChat());
        Assert.assertEquals("D", trackDimension.getDevice());
    }

    @Test
    public void test2() {
        Dimension workflowTask = ObjectBuilder.buildDimension();
        TrackDimension trackDimension = new TrackDimension(workflowTask, "T");
        Assert.assertEquals(trackDimension.getWorkflow(), workflowTask.getWorkflow());
        Assert.assertEquals("T", trackDimension.getTrack());
        Assert.assertEquals("BIZ", trackDimension.getBiz());
        Assert.assertEquals("CHAT", trackDimension.getChat());
        Assert.assertEquals("DEVICE", trackDimension.getDevice());
    }
}
