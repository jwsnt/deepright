package ai.open.right.workflow.flow.pubsub;

import ai.open.right.workflow.flow.WorkflowTask;

public interface PubSubService {

    public String sub(PubSubConfig pubSubConfig, WorkflowTask workTask) throws Exception;

    public String sub(PubSubConfig pubSubConfig, String k) throws Exception;

    public String sub(Integer timeout, String k) throws Exception;

    public String sub(String k) throws Exception;

    public void pub(PubSubConfig pubSubConfig, WorkflowTask workTask) throws Exception;

    public void pub(PubSubConfig pubSubConfig, String k, String v) throws Exception;

    public void pub(Integer expire, String k, String v) throws Exception;

    public void pub(String k, String v) throws Exception;
}
