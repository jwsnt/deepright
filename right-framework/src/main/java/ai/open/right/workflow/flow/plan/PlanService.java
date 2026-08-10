package ai.open.right.workflow.flow.plan;

import ai.open.right.workflow.flow.WorkflowTask;

public interface PlanService {

    public String plan(PlanConfig planConfig, WorkflowTask workTask) throws Exception;
}
