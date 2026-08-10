package ai.open.right.workflow.flow.llm.config;

import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.notify.Notifier;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Getter
@Setter
public class LLMTakeover extends GlobalConfig {

    protected String notifier;

    public LLMTakeover merge(LLMTakeover llmTakeover) throws Exception {
        super.merge(llmTakeover);
        if (llmTakeover != null) {
            this.notifier = StringUtils.defaultIfBlank(this.notifier, llmTakeover.notifier);
        }
        return this;
    }

    public LLMTakeover init(String notifier) {
        this.notifier = StringUtils.defaultIfBlank(this.notifier, notifier);
        return this;
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public String getNotifier() {
        return StringUtils.defaultIfBlank(this.notifier, Notifier.SOURCE);
    }
}
