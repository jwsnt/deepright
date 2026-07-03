package ai.deepright.safety;

import ai.open.right.workflow.flow.WorkflowTask;

public interface SafetyService {

    public String safety(WorkflowTask workTask) throws Exception;
}
