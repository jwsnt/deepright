package ai.open.right.workflow.flow.parallel;

import ai.open.right.workflow.flow.WorkflowTask;

public interface ParallelService {

    public String execute(ParallelConfig parallelConfig, WorkflowTask workTask) throws Exception;
}
