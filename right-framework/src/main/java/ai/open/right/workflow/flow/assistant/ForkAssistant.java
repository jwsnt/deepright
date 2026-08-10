package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.fork.ForkService;
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
// 多路分流
public class ForkAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-fork";

    protected ForkService forkService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasFork(), "Fork config can not be empty, please check config");
        if (workflowConfig.hasChain() && log.isWarnEnabled()) {
            log.warn("Fork assistant can not support `chain`");
        }
        this.forkService.fork(workflowConfig, workTask);
    }

    @ConditionalOnProperty(name = "fork.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected ForkService forkService;

        @Bean(ForkAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ForkAssistant.WORKFLOW_NAME)
        public ForkAssistant forkAssistant() throws Exception {
            ForkAssistant forkAssistant = new ForkAssistant();
            BeanUtils.copyProperties(this, forkAssistant);
            log.info("ForkAssistant inited");
            return forkAssistant;
        }
    }
}
