package ai.deepright.cli.insert;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;

import java.util.List;

public interface CliRecall {

    public List<History> recall(WorkflowTask workTask) throws Exception;
}
