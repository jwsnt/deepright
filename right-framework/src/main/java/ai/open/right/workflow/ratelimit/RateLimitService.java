package ai.open.right.workflow.ratelimit;

import ai.open.right.workflow.flow.WorkflowTask;

public interface RateLimitService {

    public void checkLimit(WorkflowTask workTask) throws Exception;
}
