package ai.open.right.workflow.flow.llm.store.history.impl;

import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryRewriter;
import lombok.Getter;
import lombok.Setter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.CollectionUtils;

import java.util.List;

@Setter
@Getter
public class BaseHistoryStore {

    @Autowired(required = false)
    // 记忆调整
    protected List<HistoryRewriter> historyRewriter;

    public HistoryPair store(Dimension dimension, HistoryPair historyPair) throws Exception {
        if (CollectionUtils.isEmpty(this.historyRewriter)) {
            return historyPair;
        }
        for (HistoryRewriter historyAdjuster : this.historyRewriter) {
            if ((historyPair = historyAdjuster.store(dimension, historyPair)) == null) {
                break;
            }
        }
        return historyPair;
    }

    public HistoryPair restore(Dimension dimension, HistoryPair historyPair) throws Exception {
        if (CollectionUtils.isEmpty(this.historyRewriter)) {
            return historyPair;
        }
        for (HistoryRewriter historyAdjuster : this.historyRewriter) {
            if ((historyPair = historyAdjuster.restore(dimension, historyPair)) == null) {
                break;
            }
        }
        return historyPair;
    }
}
