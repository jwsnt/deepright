package ai.open.right.workflow.flow.function;

import ai.open.right.workflow.flow.WorkflowTask;

public interface FunctionService {

    public Object call(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception;
}
