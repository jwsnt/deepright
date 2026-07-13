package ai.deepright.llm;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowQueue;
import lombok.Builder;
import lombok.Getter;

import java.util.concurrent.ScheduledExecutorService;

@Builder
@Getter
public class RetryConfig {

    protected ScheduledExecutorService scheduled;

    protected WorkflowQueue workflowQueue;

    protected Integer retry;

    protected Integer sleep;

    public RetryConfig check() throws Exception {
        WorkflowException.checkCondition(this.workflowQueue == null, "The workflow queue can not be empty");
        WorkflowException.checkCondition(this.scheduled == null, "The scheduled can not be empty");
        WorkflowException.checkCondition(this.retry == null, "The retry can not be empty");
        WorkflowException.checkCondition(this.sleep == null, "The sleep can not be empty");
        return this;
    }
}
