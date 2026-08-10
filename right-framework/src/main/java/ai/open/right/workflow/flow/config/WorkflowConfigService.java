package ai.open.right.workflow.flow.config;

import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.flow.WorkflowTask;

public interface WorkflowConfigService {

    public WorkflowConfig config(ConfigSearch configSearch, String workflow) throws Exception;

    // 使用指定Workflow
    public WorkflowConfig config(WorkflowTask workTask, String workflow) throws Exception;

    // 使用指定Biz+Workflow
    public WorkflowConfig config(String biz, String workflow) throws Exception;

    // 使用WorkflowTask.workflow
    public WorkflowConfig config(WorkflowTask workTask) throws Exception;
}
