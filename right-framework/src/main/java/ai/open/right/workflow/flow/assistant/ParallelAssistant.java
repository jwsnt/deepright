package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.parallel.ParallelService;
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
// 并行
public class ParallelAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-parallel";

    protected ParallelService parallelService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasParallel(), "Parallel config can not be empty, please check config");
        String query = this.parallelService.execute(workflowConfig.getParallelConfig(), workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, query);
    }

    @ConditionalOnProperty(name = "parallel.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected ParallelService parallelService;

        @Bean(ParallelAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ParallelAssistant.WORKFLOW_NAME)
        public ParallelAssistant parallelAssistant() throws Exception {
            ParallelAssistant parallelAssistant = new ParallelAssistant();
            BeanUtils.copyProperties(this, parallelAssistant);
            log.info("ParallelAssistant inited");
            return parallelAssistant;
        }
    }

}
