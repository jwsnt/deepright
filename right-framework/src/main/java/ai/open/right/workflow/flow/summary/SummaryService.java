package ai.open.right.workflow.flow.summary;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;

import java.util.List;

public interface SummaryService {

    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception;

    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception;

    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask, String append) throws Exception;

    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask) throws Exception;
}
