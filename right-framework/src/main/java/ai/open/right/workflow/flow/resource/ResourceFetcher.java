package ai.open.right.workflow.flow.resource;

import ai.open.right.workflow.flow.WorkflowTask;

public interface ResourceFetcher {

    public ResourceResponse fetch(ResourceConfig resourceConfig, WorkflowTask workTask) throws Exception;
}
