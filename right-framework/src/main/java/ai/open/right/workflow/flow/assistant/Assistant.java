package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;

public interface Assistant {

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception;

    public void config(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception;
}
