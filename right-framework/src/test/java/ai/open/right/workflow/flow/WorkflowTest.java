package ai.open.right.workflow.flow;

import org.easymock.EasyMock;
import org.junit.jupiter.api.Test;

public class WorkflowTest {

    @Test
    public void testAnonymousWorkflow() throws Exception {
        Workflow workflow = EasyMock.createMock(Workflow.class);
        WorkflowTask task = EasyMock.createMock(WorkflowTask.class);
        workflow.sync(task);
        EasyMock.expectLastCall().once();
        EasyMock.replay(workflow, task);
        workflow.sync(task);
        EasyMock.verify(workflow, task);
    }
}
