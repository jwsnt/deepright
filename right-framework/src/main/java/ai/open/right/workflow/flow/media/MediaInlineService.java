package ai.open.right.workflow.flow.media;

import ai.open.right.workflow.flow.WorkflowTask;

public interface MediaInlineService {

    public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception;
}
