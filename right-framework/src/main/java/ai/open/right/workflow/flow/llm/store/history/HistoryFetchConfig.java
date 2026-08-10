package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Getter
@Setter
public class HistoryFetchConfig extends GlobalConfig {

    public static final Integer NUMS = 100;

    protected Integer nums;

    // 摘要需要提取的记忆
    protected String scene;

    public HistoryFetchConfig merge(HistoryFetchConfig historyFetchConfig) throws Exception {
        super.merge(historyFetchConfig);
        if (historyFetchConfig != null) {
            this.scene = StringUtils.defaultIfBlank(this.scene, historyFetchConfig.scene);
            this.nums = this.nums != null ? this.nums : historyFetchConfig.nums;
        }
        return this;
    }

    public Integer getNums() {
        return this.nums != null ? this.nums : HistoryFetchConfig.NUMS;
    }
}
