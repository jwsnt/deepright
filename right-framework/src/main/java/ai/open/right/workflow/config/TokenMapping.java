package ai.open.right.workflow.config;

import ai.open.right.workflow.flow.WorkflowTask;

public interface TokenMapping {

    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception;
}
