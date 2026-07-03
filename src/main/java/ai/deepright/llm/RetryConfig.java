package ai.deepright.llm;

import ai.open.right.workflow.flow.WorkflowQueue;
import lombok.Builder;
import lombok.Getter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.Assert;

import java.util.concurrent.ScheduledExecutorService;

@Builder
@Getter
public class RetryConfig {

    protected ScheduledExecutorService scheduled;

    protected WorkflowQueue workflowQueue;

    protected Integer retry;

    protected Integer sleep;

    public RetryConfig check() throws Exception {
        Assert.notNull(this.workflowQueue, "The workflow queue can not be empty");
        Assert.notNull(this.scheduled, "The scheduled can not be empty");
        Assert.notNull(this.retry, "The retry can not be empty");
        Assert.notNull(this.sleep, "The sleep can not be empty");
        return this;
    }
}
