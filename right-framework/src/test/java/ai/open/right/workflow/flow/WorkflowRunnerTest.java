package ai.open.right.workflow.flow;

import org.easymock.EasyMock;
import org.junit.jupiter.api.Test;

public class WorkflowRunnerTest {

    @Test
    public void testAnonymousRunner() {
        WorkflowRunner runner = EasyMock.createMock(WorkflowRunner.class);
        runner.run();
        EasyMock.expectLastCall().once();
        EasyMock.replay(runner);
        runner.run();
        EasyMock.verify(runner);
    }
}
