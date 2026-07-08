package ai.deepright.llm;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.open.right.workflow.flow.WorkflowQueue;
import lombok.Builder;
import lombok.Getter;
import org.springframework.beans.factory.annotation.Autowired;

import java.util.concurrent.ScheduledExecutorService;

@Builder
@Getter
public class RetryConfig {

    protected ScheduledExecutorService scheduled;

    protected WorkflowQueue workflowQueue;

    protected Integer retry;

    protected Integer sleep;

    public RetryConfig check() throws Exception {
        WorkflowException.check(this.workflowQueue == null, "The workflow queue can not be empty", ProtocolCode.C400);
        WorkflowException.check(this.scheduled == null, "The scheduled can not be empty", ProtocolCode.C400);
        WorkflowException.check(this.retry == null, "The retry can not be empty", ProtocolCode.C400);
        WorkflowException.check(this.sleep == null, "The sleep can not be empty", ProtocolCode.C400);
        return this;
    }
}
