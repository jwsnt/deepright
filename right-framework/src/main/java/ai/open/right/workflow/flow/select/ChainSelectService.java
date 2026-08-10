package ai.open.right.workflow.flow.select;

import ai.open.right.workflow.flow.WorkflowTask;

public interface ChainSelectService {

    public String selectChain(ChainSelectConfig chainSelectConfig, WorkflowTask workTask) throws Exception;
}
