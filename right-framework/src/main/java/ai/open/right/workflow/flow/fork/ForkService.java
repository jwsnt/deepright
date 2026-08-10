package ai.open.right.workflow.flow.fork;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;

public interface ForkService {

    public void fork(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception;
}
