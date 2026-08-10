package ai.open.right.workflow.flow;

import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class WorkflowQueueTest {

    @Test
    public void testAnonymousQueue() throws Exception {
        WorkflowQueue queue = EasyMock.createMock(WorkflowQueue.class);
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        queue.put(task);
        EasyMock.expectLastCall().once();
        EasyMock.expect(queue.get()).andReturn(task).once();
        EasyMock.replay(queue, task);
        queue.put(task);
        Assertions.assertEquals(task, queue.get());
        EasyMock.verify(queue, task);
    }
}
