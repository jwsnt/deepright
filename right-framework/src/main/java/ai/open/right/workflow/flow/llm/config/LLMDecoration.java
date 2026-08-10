package ai.open.right.workflow.flow.llm.config;

import org.apache.commons.lang3.StringUtils;

import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
// LLM响应静态包装：prefix + 响应结果 + suffix
public class LLMDecoration {

    protected String prefix = "";

    protected String suffix = "";

    public LLMDecoration merge(LLMDecoration llmDecoration) throws Exception {
        if (llmDecoration != null) {
            this.prefix = StringUtils.defaultIfBlank(this.prefix, llmDecoration.prefix);
            this.suffix = StringUtils.defaultIfBlank(this.suffix, llmDecoration.suffix);
        }
        return this;
    }
}
