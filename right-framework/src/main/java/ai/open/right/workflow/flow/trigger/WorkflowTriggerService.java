package ai.open.right.workflow.flow.trigger;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;

public interface WorkflowTriggerService {
    public void before(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception;
}
