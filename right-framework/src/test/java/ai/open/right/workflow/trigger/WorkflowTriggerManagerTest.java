package ai.open.right.workflow.trigger;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.trigger.BaseTrigger;
import ai.open.right.workflow.flow.trigger.impl.WorkflowTriggerServiceImpl;
import org.junit.Test;

import java.util.Collections;

public class WorkflowTriggerManagerTest {

    @Test
    public void testTrigger() throws Exception {
        WorkflowTriggerServiceImpl workflowTriggerManager = new WorkflowTriggerServiceImpl();
        workflowTriggerManager.setTriggers(Collections.singletonMap("HELLO", new BaseTrigger()));
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setTrigger("HELLO");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTriggerManager.before(workflowConfig, workflowTask);
    }
}
