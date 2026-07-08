package ai.deepright.workflow.worktask;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.deepright.router.RouterService;
import ai.open.right.workflow.flow.WorkflowTask;

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
        WorkflowException.check(!(this.routerService.hasHeartbeat(this.workTask)), "The heartbeat workTask is closed", ProtocolCode.C400);
    }
}
