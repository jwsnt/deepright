package ai.open.right.workflow.flow.summary;

import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

@Getter
@Setter
@Builder
@Slf4j
public class SummaryPart {

    protected List<HistoryPair> pairs;

    protected String content;

    public SummaryPart init(WorkflowTask workTask, Long LastTimeline) {
        if (!CollectionUtils.isEmpty(this.pairs)) {
            // 方式不可变集合
            List<HistoryPair> pairs = new ArrayList<HistoryPair>(this.pairs);;
            Iterator<HistoryPair> iterator = pairs.iterator();
            while (iterator.hasNext()) {
                HistoryPair pair = iterator.next();
                // 如果是Assistant且Answer为空 或 User且Query为空则过滤
                if (History.ROLE_ASSISTANT.equals(pair.getRole()) && StringUtils.isEmpty(pair.getAnswer()) || History.ROLE_USER.equals(pair.getRole()) && StringUtils.isEmpty(pair.getQuery())) {
                    iterator.remove();
                    if (log.isWarnEnabled()) {
                        log.warn("The history pairs have been filtered, {}", pair);
                    }
                }
                if (StringUtils.isEmpty(pair.getConversation())) {
                    pair.setConversation(workTask.getConversation());
                }
                if (StringUtils.isEmpty(pair.getSource())) {
                    pair.setSource(SplitUtils.join(workTask));
                }
                if (StringUtils.isEmpty(pair.getModel())) {
                    pair.setModel(LLMQueryService.LLM_UNKNOW);
                }
                if (StringUtils.isEmpty(pair.getChat())) {
                    pair.setChat(workTask.getChat());
                }
                if (StringUtils.isEmpty(pair.getApi())) {
                    pair.setApi(LLMQueryService.LLM_UNKNOW);
                }
                if (pair.getCreated() == null) {
                    pair.setCreated(LastTimeline);
                }
            }
            this.pairs = pairs;
        }
        return this;
    }

    public Boolean hasContent() {
        return !StringUtils.isEmpty(this.content);
    }

    public Boolean hasPairs() {
        return !CollectionUtils.isEmpty(this.pairs);
    }
}
