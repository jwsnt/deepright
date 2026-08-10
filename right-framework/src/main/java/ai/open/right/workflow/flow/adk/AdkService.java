package ai.open.right.workflow.flow.adk;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;

public interface AdkService {

    public String execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception;
}
