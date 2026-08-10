package ai.open.right.workflow.condition;

import lombok.Builder;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;

@Builder
@Getter
@Slf4j
public class Condition {

    public static final Condition FALSE = Condition.builder().condition(false).build();

    protected Boolean condition;

    protected String content;

    public Condition print() throws Exception {
        if (log.isInfoEnabled()) {
            log.info("Condition content={}", this.content);
        }
        return this;
    }
}