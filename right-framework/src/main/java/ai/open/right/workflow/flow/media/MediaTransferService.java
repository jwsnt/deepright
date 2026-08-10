package ai.open.right.workflow.flow.media;

import ai.open.right.workflow.flow.WorkflowTask;

import java.util.List;

public interface MediaTransferService {

    public void transfer(MediaConfig mediaConfig, WorkflowTask workTask, List<MediaContext> mediaContext) throws Exception;

    public void transfer(WorkflowTask workTask, List<MediaContext> mediaContext) throws Exception;
}
