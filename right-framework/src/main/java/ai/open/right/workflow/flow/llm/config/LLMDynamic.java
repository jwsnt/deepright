package ai.open.right.workflow.flow.llm.config;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
@ToString
// 动态System Prompt（@See DyPromptService）
public class LLMDynamic {

    // 失败时是否终止，如果为True则System Prompt等于Query
    protected Boolean stopOnFailed;

    // 调用下游思考链（Workflow）超时
    protected Integer timeout;

    // 生成动态System Prompt思考结果的通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // 使用的思考链（Workflow）
    protected String dynamic;

    public LLMDynamic merge(LLMDynamic llmDynamic) throws Exception {
        if (llmDynamic != null) {
            this.stopOnFailed = this.stopOnFailed != null ? this.stopOnFailed : llmDynamic.stopOnFailed;
            this.notifier = StringUtils.defaultIfBlank(this.notifier, llmDynamic.notifier);
            this.dynamic = StringUtils.defaultIfBlank(this.dynamic, llmDynamic.dynamic);
            this.timeout = this.timeout != null ? this.timeout : llmDynamic.timeout;
        }
        return this;
    }

    public LLMDynamic init(String notifier) {
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        return this;
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }

    public Boolean getStopOnFailed() {
        return this.stopOnFailed != null ? this.stopOnFailed : true;
    }
}
