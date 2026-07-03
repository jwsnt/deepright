package ai.deepright.workflow.worktask;

import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class ResetStateWorkTask extends MarkQueryWorkTask {

    protected String original;

    protected String initial;

    public ResetStateWorkTask(WorkflowTask workTask, String query) {
        super(workTask, true, query, System.currentTimeMillis());
        this.original = workTask.getOriginal();
        this.initial = query;
    }
}
