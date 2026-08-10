package ai.open.right.workflow.flow;

import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class WorkflowTaskTest {

    @Test
    public void testAnonymousTask() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        EasyMock.expect(task.getWorkflow()).andReturn("test-workflow").anyTimes();
        EasyMock.replay(task);
        Assertions.assertEquals("test-workflow", task.getWorkflow());
        EasyMock.verify(task);
    }

    @Test
    public void testMarkQueryResetQuery_contract() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        task.markQuery();
        EasyMock.expectLastCall().once();
        task.resetQuery();
        EasyMock.expectLastCall().once();
        EasyMock.replay(task);
        task.markQuery();
        task.resetQuery();
        EasyMock.verify(task);
    }

    @Test
    public void testGetCreated_contract() {
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        Long created = 12345L;
        EasyMock.expect(task.getCreated()).andReturn(created).anyTimes();
        EasyMock.replay(task);
        Assertions.assertEquals(created, task.getCreated());
        EasyMock.verify(task);
    }
}
