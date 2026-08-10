package ai.open.right.workflow.flow.llm.rag;

import lombok.Getter;
import lombok.Setter;
import org.springframework.util.StringUtils;

@Setter
@Getter
public class RagOrchestrator {

    protected String before;

    protected String after;

    public Boolean hasBefore() {
        return StringUtils.hasText(this.before);
    }

    public Boolean hasAfter() {
        return StringUtils.hasText(this.after);
    }
}

