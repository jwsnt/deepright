package ai.deepright.skills;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.SkillsAssistant;
import ai.open.right.workflow.notify.Notifier;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

import java.util.ArrayList;
import java.util.List;

@Getter
@Setter
@Slf4j
public class HintSkillAssistant extends SkillsAssistant {

    public static final String LANG_KEY_SKILLS_HINT_RECALL = "skills.hint.recall";

    public static final String LANG_KEY_SKILLS_HINT_INIT = "skills.hint.init";

    public static final String KEY = "skills.hint";

    protected Boolean hint;

    @Override
    protected String buildSkill(WorkflowTask workTask, SkillOperation skillOperation) throws Exception {
        String content = super.buildSkill(workTask, skillOperation);
        this.source(workTask, skillOperation.getSkill(), content);
        return content;
    }

    protected void source(WorkflowTask workTask, String skill, String content) throws Exception {
        if (!FeatureFlag.isSilent(workTask) && this.hint) {
            List<String> skills = workTask.getMetadata(HintSkillAssistant.KEY, List.class);
            skills = skills != null ? skills : new ArrayList<String>();
            if (skills.contains(skill)) {
                this.notify(workTask, CliPrinter.process(HintSkillAssistant.KEY), Notifier.SOURCE, XmlResourceLang.get(HintSkillAssistant.LANG_KEY_SKILLS_HINT_RECALL).replace("#skill", skill));
            } else {
                this.notify(workTask, CliPrinter.process(HintSkillAssistant.KEY), Notifier.SOURCE, XmlResourceLang.get(HintSkillAssistant.LANG_KEY_SKILLS_HINT_INIT).replace("#skill", skill));
                skills.add(skill);
            }
            workTask.putMetadata(HintSkillAssistant.KEY, skills);
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class HintInitConfig extends InitConfig {

        @Value("${skills.hint:true}")
        protected Boolean hint;

        @Override
        @Bean(SkillsAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = SkillsAssistant.WORKFLOW_NAME)
        public SkillsAssistant skillsAssistant() throws Exception {
            HintSkillAssistant skillsAssistant = new HintSkillAssistant();
            BeanUtils.copyProperties(this, skillsAssistant);
            HintSkillAssistant.log.info("HintSkillAssistant inited");
            return skillsAssistant;
        }
    }
}
