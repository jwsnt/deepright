package ai.open.right.workflow.flow.llm.store.history;

public interface HistoryTruncate {

    public String truncate(String histories) throws Exception;
}
