package ai.open.right.workflow.flow.competition;

import ai.open.right.workflow.flow.WorkflowTask;

public interface CompetitionService {

    public String compete(CompetitionConfig competitionConfig, WorkflowTask workTask) throws Exception;
}
