package ai.open.right.workflow.flow.config;

import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
// 公共Timeout配置
public class TimeoutConfig extends GlobalConfig{

    // 调用条件分支思考链（Workflow）的超时
    protected Integer timeout4Condition;

    // 调用纠正结果思考链（Workflow）的超时（例如Script脚本时）
    protected Integer timeout4Corrector;

    // 调用下游思考链（Workflow）的超时
    protected Integer timeout4Service;

    // 调用LLM时的超时
    protected Integer timeout4Llm;

    // 调用超时
    protected Integer timeout;

    public TimeoutConfig merge(TimeoutConfig timeoutConfig) throws Exception {
        super.merge(timeoutConfig);
        if (timeoutConfig != null) {
            this.timeout4Condition = this.timeout4Condition != null ? this.timeout4Condition : timeoutConfig.timeout4Condition;
            this.timeout4Corrector = this.timeout4Corrector != null ? this.timeout4Corrector : timeoutConfig.timeout4Corrector;
            this.timeout4Service = this.timeout4Service != null ? this.timeout4Service : timeoutConfig.timeout4Service;
            this.timeout4Llm = this.timeout4Llm != null ? this.timeout4Llm : timeoutConfig.timeout4Llm;
            this.timeout = this.timeout != null ? this.timeout : timeoutConfig.timeout;
        }
        return this;
    }

    public Integer getTimeout4Condition(Integer timeout4condition) {
        return this.timeout4Condition != null ? this.timeout4Condition : timeout4condition;
    }

    public Integer getTimeout4Corrector(Integer timeout4corrector) {
        return this.timeout4Corrector != null ? this.timeout4Corrector : timeout4corrector;
    }

    public Integer getTimeout4Service(Integer timeout4service) {
        return this.timeout4Service != null ? this.timeout4Service : timeout4service;
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        return this.timeout4Llm != null ? this.timeout4Llm : timeout4llm;
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }
}
