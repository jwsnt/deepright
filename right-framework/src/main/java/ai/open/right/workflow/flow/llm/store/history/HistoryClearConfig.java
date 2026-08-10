package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;

import java.util.Arrays;
import java.util.List;

@Setter
@Getter
public class HistoryClearConfig extends GlobalConfig {

    // 需要清理的记忆Key
    protected List<String> repositories;

    // 清理偏移
    protected Integer offset;

    public HistoryClearConfig merge(HistoryClearConfig historyClearConfig) throws Exception {
        super.merge(historyClearConfig);
        if (historyClearConfig != null) {
            this.repositories = CollectionsUtils.merge(this.repositories, historyClearConfig.repositories);
            this.offset = this.offset != null ? this.offset : historyClearConfig.offset;
        }
        return this;
    }

    public List<String> getRepositories(String repository) {
        return this.repositories != null ? this.repositories : Arrays.asList(repository);
    }

    public Long getOffset(Long now) {
        return this.offset != null ? now - this.offset : now;
    }
}
