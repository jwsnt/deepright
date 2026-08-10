package ai.open.right.workflow.flow.track;

import ai.open.right.workflow.flow.WorkflowTask;

import java.util.List;

public interface TrackChatService {

    // 恢复Chat + Device的终端用户可见的会话内容
    public List<TrackChatBody> restore(WorkflowTask workTask) throws Exception;

    // 存储Chat + Device的终端用户可见的会话内容
    public void store(TrackChat trackChat) throws Exception;
}
