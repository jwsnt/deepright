package ai.open.right.workflow.flow.assistant;

import ai.open.right.protocol.Protocol;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.iteration.IterationService;
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

@Slf4j
@Setter
@Getter
// 迭代优化
public class IterationAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-iteration";

    protected IterationService iterationService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasIteration(), "Iteration config can not be empty, please check config");
        String iterationContent = this.iterationService.iterate(workflowConfig.getIterationConfig(), workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, Protocol.CHAT, iterationContent);
    }

    @ConditionalOnProperty(name = "iteration.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected IterationService iterationService;

        @Bean(IterationAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = IterationAssistant.WORKFLOW_NAME)
        public IterationAssistant iterationAssistant() throws Exception {
            IterationAssistant iterationAssistant = new IterationAssistant();
            BeanUtils.copyProperties(this, iterationAssistant);
            log.info("IterationAssistant inited");
            return iterationAssistant;
        }
    }
}
