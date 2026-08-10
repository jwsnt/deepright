package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.skill.SkillsFetcher;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Setter
@Getter
@Slf4j
public class SkillsAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-skills";

    protected SkillsFetcher skillFetcher;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        SkillOperation skillOperation = this.buildPath(workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildSkill(workTask, skillOperation));
    }

    protected String buildSkill(WorkflowTask workTask, SkillOperation skillOperation) throws Exception {
        String content = this.skillFetcher.fetchResource(workTask, skillOperation.getSkill(), skillOperation.getResource());
        if (log.isDebugEnabled()) {
            log.debug("Skills skill={}, resource={}, content={}", skillOperation.getSkill(), skillOperation.getResource(), content);
        }
        return content;
    }

    protected SkillOperation buildPath(WorkflowTask workTask) throws Exception {
        SkillOperation skillOperation = workTask.getObjectQuery(SkillOperation.class);
        Assert.hasText(skillOperation.getSkill(), "Skill can not be empty: " + workTask.getQuery());
        return skillOperation;
    }

    @Getter
    @Setter
    public static class SkillOperation {

        protected String resource;

        protected String skill;

        @JsonProperty("why_do_this")
        protected String why;
    }

    @ConditionalOnProperty(name = "skills.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected SkillsFetcher skillFetcher;

        @Bean(SkillsAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = SkillsAssistant.WORKFLOW_NAME)
        public SkillsAssistant skillsAssistant() throws Exception {
            SkillsAssistant skillAssistant = new SkillsAssistant();
            BeanUtils.copyProperties(this, skillAssistant);
            log.info("SkillsAssistant inited");
            return skillAssistant;
        }
    }
}
