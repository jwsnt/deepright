package ai.open.right.workflow.flow.plan;

import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
@ToString
public class PlanConfig extends GlobalConfig {

    public static final Integer MIN = 20;

    @JsonProperty("iteration")
    protected IterationConfig iterationConfig;

    // 是否开启多论会话记忆
    protected Boolean containHistories;

    @JsonProperty("notifier")
    // Plan通知方式
    protected PlanNotifier notifier;

    protected LLMConfig llmConfig;

    // 调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // 调用异常结束的总结的思考链（Workflow）
    protected String exception;

    // 调用正常结束的总结的思考链（Workflow）
    protected String summary;

    // 调用Plan拆分任务的思考链（Workflow）
    protected String plan;

    public PlanConfig merge(PlanConfig planConfig) throws Exception {
        super.merge(planConfig);
        if (planConfig != null) {
            this.iterationConfig = this.iterationConfig != null ? this.iterationConfig.merge(planConfig.iterationConfig) : planConfig.iterationConfig;
            this.containHistories = this.containHistories != null ? this.containHistories : planConfig.containHistories;
            this.notifier = this.notifier != null ? this.notifier.merge(planConfig.notifier) : planConfig.notifier;
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : planConfig.timeout4Llm;
            this.exception = StringUtils.defaultIfBlank(this.exception, planConfig.exception);
            this.summary = StringUtils.defaultIfBlank(this.summary, planConfig.summary);
            this.plan = StringUtils.defaultIfBlank(this.plan, planConfig.plan);
        }
        return this;
    }

    public PlanConfig init(LLMConfig llmConfig) {
        if (this.hasIteration()) {
            this.getIterationConfig().init(llmConfig);
        }
        this.llmConfig = this.llmConfig != null ? this.llmConfig : llmConfig;
        return this;
    }

    public PlanConfig init(String notifier) {
        if (!this.hasNotifier()) {
            this.setNotifier(new PlanNotifier().init(notifier));
        } else {
            this.getNotifier().init(notifier);
        }
        // 初始化Iteration
        if (this.hasIteration()) {
            this.getIterationConfig().init(notifier);
        }
        return this;
    }

    public Boolean getContainHistories() {
        return this.containHistories != null ? this.containHistories : false;
    }

    public Integer getTimeout4Llm(Integer timeout) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout;
    }

    public Boolean hasNotifierWithException() {
        return this.hasNotifier() && this.getNotifier().hasException();
    }

    public Boolean hasNotifierWithSummary() {
        return this.hasNotifier() && this.getNotifier().hasSummary();
    }

    public Boolean hasNotifierWithPlan() {
        return this.hasNotifier() && this.getNotifier().hasPlan();
    }

    public Boolean hasException() {
        return !StringUtils.isEmpty(this.exception);
    }

    public Boolean hasIteration() {
        return this.iterationConfig != null;
    }

    public Boolean hasNotifier() {
        return this.notifier != null;
    }

    public Boolean hasSummary() {
        return !StringUtils.isEmpty(this.summary);
    }

    public Boolean hasPlan() {
        return !StringUtils.isEmpty(this.plan);
    }
}
