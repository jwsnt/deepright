package ai.open.right.workflow.flow;

public interface Workflow {

    public void async(WorkflowTask workTask) throws Exception;

    public void sync(WorkflowTask workTask) throws Exception;
}
