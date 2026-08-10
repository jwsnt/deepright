package ai.open.right.workflow.flow.block;

import ai.open.right.workflow.flow.WorkflowTask;

public interface BlockService {

    public void submit(String biz, String chat, String device, WorkflowTask workTask, Long timestamp) throws Exception;

    public void submit(String biz, String chat, WorkflowTask workTask, Long timestamp) throws Exception;

    public void submit(String biz, String chat, String device, WorkflowTask workTask) throws Exception;

    public void submit(String biz, String chat, WorkflowTask workTask) throws Exception;

    public void submit(WorkflowTask workTask, Long timestamp) throws Exception;

    public void submit(WorkflowTask workTask) throws Exception;

    public void block(String biz, String chat, String device, WorkflowTask workTask) throws Exception;

    public void block(String biz, String chat, WorkflowTask workTask) throws Exception;

    public void block(WorkflowTask workTask) throws Exception;
}
