package ai.open.right.workflow.flow.parallel;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.List;

@Setter
@Getter
public class ParallelConfig extends GlobalConfig {

    @JsonProperty("flows")
    // 并行分支
    protected List<ParallelFlow> parallelFlow;

    // 调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // 通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    public ParallelConfig merge(ParallelConfig parallelConfig) throws Exception {
        super.merge(parallelConfig);
        if (parallelConfig != null) {
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : parallelConfig.timeout4Llm;
            this.parallelFlow = CollectionsUtils.merge(this.parallelFlow,parallelConfig.parallelFlow);
            this.notifier = StringUtils.defaultIfBlank(this.notifier, parallelConfig.notifier);
        }
        return this;
    }

    public ParallelConfig init(String notifier) {
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        return this;
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4llm;
    }

    public Boolean hasParallelFlow() {
        return !CollectionUtils.isEmpty(this.parallelFlow);
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }
}
