package ai.open.right.workflow.flow.media;

import ai.open.right.workflow.flow.WorkflowTask;

import java.util.List;

public interface MediaPackageService {

    public List<MediaPackage> pack(MediaConfig mediaConfig, WorkflowTask workTask) throws Exception;

}
