package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.WorkflowTask;

public interface ProviderRequestModel {

    // 默认，兼容
    public static final String DEF = "DEF";

    public String getModel(WorkflowTask workTask) throws Exception;
}
