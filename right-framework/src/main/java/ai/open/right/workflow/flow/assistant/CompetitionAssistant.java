package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.competition.CompetitionService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
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
// 多路竞争思考链（Workflow）选择一个
public class CompetitionAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-competition";

    protected CompetitionService competitionService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasCompetition(), "Competition config can not be empty, please check config");
        // 指定下一个思考链（Workflow）
        workflowConfig.setChain(this.competitionService.compete(workflowConfig.getCompetitionConfig(), workTask));
        this.chainOr2Endpoint(workflowConfig, workTask, workTask.getQuery());
    }

    @ConditionalOnProperty(name = "competition.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected CompetitionService competitionService;

        @Bean(CompetitionAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = CompetitionAssistant.WORKFLOW_NAME)
        public CompetitionAssistant competitionAssistant() throws Exception {
            CompetitionAssistant competitionAssistant = new CompetitionAssistant();
            BeanUtils.copyProperties(this, competitionAssistant);
            log.info("CompetitionAssistant inited");
            return competitionAssistant;
        }
    }
}
