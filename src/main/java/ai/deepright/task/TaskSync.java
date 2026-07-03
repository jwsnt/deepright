package ai.deepright.task;

import ai.deepright.router.RouterDevice;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Builder;
import lombok.Getter;

@Getter
@Builder
public class TaskSync {

    protected SyncWorkflowTask syncWorkflowTask;

    protected RouterDevice sourceDevice;

    protected RouterDevice targetDevice;

    protected TaskData taskData;

    protected String error;
}