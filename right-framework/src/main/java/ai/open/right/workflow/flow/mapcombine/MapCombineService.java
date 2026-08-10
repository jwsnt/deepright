package ai.open.right.workflow.flow.mapcombine;

import ai.open.right.workflow.flow.WorkflowTask;

public interface MapCombineService {

    public String execute(MapCombineConfig mapCombineConfig, WorkflowTask workTask) throws Exception;
}
