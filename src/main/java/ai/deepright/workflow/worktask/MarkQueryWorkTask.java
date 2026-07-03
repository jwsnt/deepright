package ai.deepright.workflow.worktask;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.WorkflowTaskWrap;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class MarkQueryWorkTask extends WorkflowTaskWrap {

    protected String markQuery;

    protected String query;

    public MarkQueryWorkTask(WorkflowTask workTask, Boolean closeable, String query, Long created) {
        super(workTask, closeable, created);
        this.query = query;
    }

    public MarkQueryWorkTask(WorkflowTask workTask, Boolean closeable, String query) {
        this(workTask, closeable, query, workTask.getCreated());
    }

    public MarkQueryWorkTask(WorkflowTask workTask, Boolean closeable) {
        this(workTask, closeable, workTask.getQuery());
    }

    public MarkQueryWorkTask(WorkflowTask workTask, String query) {
        this(workTask, true, query);
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return JsonUtils.read(this.query, clazz);
    }

    @Override
    public void resetQuery() {
        this.query = this.markQuery;
    }

    @Override
    public void markQuery() {
        this.markQuery = this.query;
    }
}
