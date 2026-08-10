package ai.open.right.workflow.flow.script;

import ai.open.right.workflow.flow.WorkflowTask;

public interface ScriptService {

    public String run(ScriptConfig scriptConfig, WorkflowTask workTask) throws Exception;
}
