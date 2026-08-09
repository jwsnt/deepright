package ai.deepright.llm.token;

import ai.open.right.workflow.flow.WorkflowTask;

public interface TokenSource {

    public String source(WorkflowTask workTask) throws Exception;
}
