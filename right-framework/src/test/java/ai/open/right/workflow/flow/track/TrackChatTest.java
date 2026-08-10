package ai.open.right.workflow.flow.track;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import org.junit.Assert;
import org.junit.Test;

public class TrackChatTest {

    @Test
    public void test() {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        TrackChatBody body = new TrackChatBody();
        TrackChat trackChat = TrackChat.builder().build();
        trackChat.setTrackChatBody(body);
        trackChat.setDimension(workflowTask);
        Assert.assertEquals(workflowTask, trackChat.getDimension());
        Assert.assertEquals(body, trackChat.getTrackChatBody());
    }
}
