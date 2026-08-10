package ai.open.right.workflow.config;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.*;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

@Getter
@Setter
@Builder
@ToString
@AllArgsConstructor
public class PromptSearch {

    @JsonIgnore
    protected WorkflowTask workTask;

    @JsonIgnore
    protected LLMConfig llmConfig;

    // 通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // Prompt配置文件名称
    protected String prompt;

    protected String biz;

    public PromptSearch() {

    }

    public String getLanguage() {
        return this.workTask.getUserContext().getLanguage();
    }

    public String getDevice() {
        return this.workTask.getUserContext().getDevice();
    }

    public String getPrompt() {
        // 指定了特定Prompt配置文件名称 > 指定了LLMConfig的Prompt的配置名称 > 默认使用Workflow名称
        return StringUtils.hasText(this.prompt) ? this.prompt : (StringUtils.hasText(this.llmConfig.getPrompt()) ? this.llmConfig.getPrompt() : this.workTask.getWorkflow());
    }

    public String getBiz() {
        // 指定了特定Biz > 指定了LLMConfig的Biz名称
        return StringUtils.hasText(this.biz) ? this.biz : this.workTask.getBiz();
    }

    public Boolean hasNotifier() {
        return StringUtils.hasText(this.notifier);
    }

    public static class PromptSearchChecker {

        public static void check(PromptSearch search) {
            Assert.hasText(search.getBiz(), "Biz can not be empty");
            Assert.hasText(search.getDevice(), "Device can not be empty");
            Assert.hasText(search.getPrompt(), "Workflow can not be empty");
            Assert.hasText(search.getLanguage(), "Language can not be empty");
        }
    }
}
