package ai.open.right.workflow.flow.llm.signal;

import ai.open.right.workflow.flow.config.WorkflowConfig;

public interface SignalFactory {

    public SignalStream signal(WorkflowConfig workflowConfig);
}
