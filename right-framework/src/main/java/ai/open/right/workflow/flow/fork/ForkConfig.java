package ai.open.right.workflow.flow.fork;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Setter
@Getter
public class ForkConfig extends GlobalConfig {

    // Fork分支
    protected List<ForkTarget> target;

    // 任一失败，是否终止
    protected Boolean stopOnFailed;

    // 调用超时（覆盖默认超时）
    protected Integer timeout;

    public ForkConfig merge(ForkConfig forkConfig) throws Exception {
        super.merge(forkConfig);
        if (forkConfig != null) {
            this.stopOnFailed = this.stopOnFailed != null ? this.stopOnFailed : forkConfig.stopOnFailed;
            this.timeout = this.timeout != null ? this.timeout : forkConfig.timeout;
            this.target = CollectionsUtils.merge(this.target, forkConfig.target);
        }
        return this;
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }

    public Boolean getStopOnFailed() {
        return this.stopOnFailed != null ? this.stopOnFailed : false;
    }
}
