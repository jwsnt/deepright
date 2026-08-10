package ai.open.right.workflow.flow;

public interface WorkflowQueue {

    public void put(WorkflowTask workTask) throws Exception;

    public WorkflowTask get() throws Exception;
}
