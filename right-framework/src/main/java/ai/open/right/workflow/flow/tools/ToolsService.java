package ai.open.right.workflow.flow.tools;

import ai.open.right.workflow.flow.WorkflowTask;

public interface ToolsService {

    public String execute(ToolsConfig toolsConfig, WorkflowTask workTask) throws Exception;
}
