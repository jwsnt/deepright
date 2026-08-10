package ai.open.right.workflow.flow.pubsub;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;

public interface PubSubFormatter {

    public Segment format(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, Segment segment) throws Exception;
}
