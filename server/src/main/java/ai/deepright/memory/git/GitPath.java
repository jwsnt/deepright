package ai.deepright.memory.git;

import ai.deepright.cli.CliPubData;
import ai.open.right.workflow.flow.WorkflowTask;

public interface GitPath {

    public CliPubData buildInitFile(WorkflowTask workTask) throws Exception;

    public String buildGitPath(WorkflowTask workTask) throws Exception;

    public String buildGitData(WorkflowTask workTask) throws Exception;

    public String buildGitApp(WorkflowTask workTask) throws Exception;
}
