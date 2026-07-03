package ai.deepright.cli;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.deepright.router.RouterDevice;

public interface CliTransfer {

    // 传递文件
    public CliTransferData transfer(WorkflowTask workTask, RouterDevice source, RouterDevice target, String path, String why) throws Exception;
}
