package ai.open.right.workflow.flow.iteration;

import ai.open.right.workflow.flow.WorkflowTask;

public interface IterationService {

    public String iterate(IterationConfig iterationConfig, WorkflowTask workTask) throws Exception;
}
