package ai.deepright.cli.insert;

import ai.open.right.workflow.flow.WorkflowTask;

import java.util.List;

public interface CliInsertService {

    public void insert(WorkflowTask workTask, List<CliInsert> inserts) throws Exception;
}
