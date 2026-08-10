package ai.open.right.workflow.flow.iteration;

import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class IterationConfig extends GlobalConfig {

    public static final Integer MIN_TIMES = 5;

    @JsonProperty("notifier")
    // 迭代思考通知方式（Localhost/Endpoint/Source）
    protected IterationNotifier notifier;

    // 是否开启多论会话记忆
    protected Boolean containHistories;

    // 是否追踪Fun Call
    protected Boolean funCallTrack;

    // LLM Config引用
    protected LLMConfig llmConfig;

    // 用于反思的思考链（Workflow）
    protected String refection;

    // 用于条件判断的思考链（Workflow）
    protected String condition;

    // 用于执行反思的思考链（Workflow）
    protected String processor;

    // 调用下游思考链（Workflow）超时（覆盖默认超时）
    protected Integer timeout;

    // 最大迭代次数
    protected Integer times;

    public IterationConfig merge(IterationConfig iterationConfig) throws Exception {
        super.merge(iterationConfig);
        if (iterationConfig != null) {
            this.llmConfig = this.llmConfig != null ? this.llmConfig.merge(iterationConfig.llmConfig) : iterationConfig.llmConfig;
            this.notifier = this.notifier != null ? this.notifier.merge(iterationConfig.notifier) : iterationConfig.notifier;
            this.containHistories = this.containHistories != null ? this.containHistories : iterationConfig.containHistories;
            this.funCallTrack = this.funCallTrack != null ? this.funCallTrack : iterationConfig.funCallTrack;
            this.refection = StringUtils.defaultIfBlank(this.refection, iterationConfig.refection);
            this.condition = StringUtils.defaultIfBlank(this.condition, iterationConfig.condition);
            this.processor = StringUtils.defaultIfBlank(this.processor, iterationConfig.processor);
            this.timeout = this.timeout != null ? this.timeout : iterationConfig.timeout;
            this.times = this.times != null ? this.times : iterationConfig.times;
        }
        return this;
    }

    public IterationConfig init(LLMConfig llmConfig) {
        this.llmConfig = this.llmConfig != null ? this.llmConfig : llmConfig;
        return this;
    }

    public IterationConfig init(String notifier) {
        if (!this.hasNotifier()) {
            this.setNotifier(new IterationNotifier().init(notifier));
        } else {
            this.getNotifier().init(notifier);
        }
        return this;
    }

    public Boolean getContainHistories() {
        return this.containHistories != null ? this.containHistories : false;
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout == null ? timeout : Math.min(this.timeout, timeout);
    }

    public Integer getTimes() {
        return this.times != null ? this.times : IterationConfig.MIN_TIMES;
    }

    public Boolean hasCondition() {
        return !StringUtils.isEmpty(this.condition);
    }

    public Boolean hasProcessor() {
        return !StringUtils.isEmpty(this.processor);
    }

    public Boolean hasRefection() {
        return !StringUtils.isEmpty(this.refection);
    }

    public Boolean hasFunCallTrack() {
        return this.funCallTrack != null && this.funCallTrack;
    }

    public Boolean hasNotifierWithRefection() {
        return this.hasNotifier() && this.getNotifier().hasRefection();
    }

    public Boolean hasNotifierWithProcessor() {
        return this.hasNotifier() && this.getNotifier().hasProcessor();
    }

    public Boolean hasNotifier() {
        return this.notifier != null;
    }
}
