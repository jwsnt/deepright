package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.workflow.flow.llm.store.Dimension;

public interface HistoryRewriter {

    public HistoryPair store(Dimension dimension, HistoryPair historyPair);

    public HistoryPair restore(Dimension dimension, HistoryPair historyPair);
}
