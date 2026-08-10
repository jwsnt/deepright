package ai.open.right.workflow.flow.competition;

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
public class CompetitionConfig extends GlobalConfig {

    @JsonProperty("conditions")
    // 条件分支
    protected List<ConditionConfig> conditionConfigs;

    // 条件分支失败时是否终止思考链（Workflow）
    protected Boolean stopOnFailed;

    // 条件分支调用超时（覆盖默认超时）
    protected Integer timeout;

    // 所有条件分支都不满足时，使用该配置（兜底）
    public String dynamic;

    public CompetitionConfig merge(CompetitionConfig competitionConfig) throws Exception {
        super.merge(competitionConfig);
        if (competitionConfig != null) {
            this.conditionConfigs = CollectionsUtils.merge(this.conditionConfigs,competitionConfig.conditionConfigs);
            this.stopOnFailed = this.stopOnFailed != null ? this.stopOnFailed : competitionConfig.stopOnFailed;
            this.dynamic = StringUtils.defaultIfBlank(this.dynamic, competitionConfig.dynamic);
            this.timeout = this.timeout != null ? this.timeout : competitionConfig.timeout;
        }
        return this;
    }

    public Boolean getStopOnFailed() {
        return this.stopOnFailed != null ? this.stopOnFailed : false;
    }

    public Boolean hasConditions() {
        return !CollectionUtils.isEmpty(this.conditionConfigs);
    }

    public Boolean hasTarget() {
        return !StringUtils.isEmpty(this.dynamic);
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }
}
