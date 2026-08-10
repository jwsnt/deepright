package ai.open.right.workflow.config;

import lombok.Getter;
import lombok.Setter;
import org.springframework.util.Assert;

@Setter
@Getter
public class Prompt {

    protected final String workflow;

    protected final String content;

    protected final String biz;

    public Prompt(String biz, String workflow, String content) {
        this.workflow = workflow;
        this.content = content;
        this.biz = biz;
    }

    public static class PromptChecker {

        public static void check(Prompt prompt) {
            Assert.hasText(prompt.getWorkflow(), "Workflow can not be empty");
            Assert.hasText(prompt.getBiz(), "Biz can not be empty");
        }
    }
}
