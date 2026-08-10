package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.summary.SummaryConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
public class HistoryConfig extends GlobalConfig {

    public static final Integer FETCH = 100;

    @JsonProperty("fetch")
    protected HistoryFetchConfig fetchConfig;

    @JsonProperty("clear")
    // 清理模式
    protected HistoryClearConfig clearConfig;

    @JsonProperty("summary")
    // 摘要模式
    protected SummaryConfig summaryConfig;

    public HistoryConfig merge(HistoryConfig historyConfig) throws Exception {
        super.merge(historyConfig);
        if (historyConfig != null) {
            this.summaryConfig = this.summaryConfig != null ? this.summaryConfig.merge(historyConfig.summaryConfig) : historyConfig.summaryConfig;
            this.clearConfig = this.clearConfig != null ? this.clearConfig.merge(historyConfig.clearConfig) : historyConfig.clearConfig;
            this.fetchConfig = this.fetchConfig != null ? this.fetchConfig.merge(historyConfig.fetchConfig) : historyConfig.fetchConfig;
        }
        return this;
    }

    public Boolean hasSummary() {
        return this.summaryConfig != null;
    }

    public Boolean hasClear() {
        return this.clearConfig != null;
    }

    public Boolean hasFetch() {
        return this.fetchConfig != null;
    }

    // 只要要包含Clear或Summary
    public Boolean isValid() {
        return this.hasClear() || this.hasSummary() || this.hasFetch();
    }
}
