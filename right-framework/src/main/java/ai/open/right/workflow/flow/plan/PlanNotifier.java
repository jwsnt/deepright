package ai.open.right.workflow.flow.plan;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class PlanNotifier extends GlobalConfig {

    // 异常结束的通知方式
    protected String exception;

    // 正常结束的通知方式
    protected String summary;

    // 计划拆分的通知方式
    protected String plan;

    public PlanNotifier merge(PlanNotifier planNotifier) throws Exception {
        super.merge(planNotifier);
        if (planNotifier != null) {
            this.exception = StringUtils.defaultIfBlank(this.exception, planNotifier.exception);
            this.summary = StringUtils.defaultIfBlank(this.summary, planNotifier.summary);
            this.plan = StringUtils.defaultIfBlank(this.plan, planNotifier.plan);
        }
        return this;
    }

    public PlanNotifier init(String notifier) {
        this.exception = StringUtils.defaultString(this.exception, notifier);
        this.summary = StringUtils.defaultString(this.summary, notifier);
        this.plan = StringUtils.defaultString(this.plan, notifier);
        return this;
    }

    public Boolean hasException() {
        return !StringUtils.isEmpty(this.exception);
    }

    public Boolean hasSummary() {
        return !StringUtils.isEmpty(this.summary);
    }

    public Boolean hasPlan() {
        return !StringUtils.isEmpty(this.plan);
    }
}
