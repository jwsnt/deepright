package ai.open.right.workflow.skill;

import lombok.Builder;
import lombok.Getter;

import java.util.Map;

@Getter
@Builder
public class Skills {

    protected Map<String, SkillMetadata> skills;

    protected String usage;

    public Skills copy() throws Exception {
        return Skills.builder()
                .skills(this.skills)
                .usage(this.usage)
                .build();
    }
}
