package ai.deepright.task;

import ai.open.right.workflow.flow.WorkflowTask;

public interface TaskResult {

    public String buildAnswer(WorkflowTask workTask, TaskSync syncTask) throws Exception;

    public String buildError(WorkflowTask workTask, TaskSync syncTask) throws Exception;
}