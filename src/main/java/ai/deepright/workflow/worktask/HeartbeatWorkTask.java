package ai.deepright.workflow.worktask;

import ai.deepright.router.RouterService;
import ai.open.right.workflow.flow.WorkflowTask;
import org.springframework.util.Assert;

public class HeartbeatWorkTask extends MarkQueryWorkTask {

    protected final RouterService routerService;

    public HeartbeatWorkTask(RouterService routerService, WorkflowTask workTask, Boolean closeable) {
        super(workTask, closeable);
        this.routerService = routerService;
    }

    @Override
    public Boolean isClosed() throws Exception {
        return super.isClosed() && !this.routerService.hasHeartbeat(this.workTask);
    }

    @Override
    public void close() throws Exception {
        super.close();
        Assert.isTrue(this.routerService.hasHeartbeat(this.workTask), "The heartbeat workTask is closed");
    }
}
