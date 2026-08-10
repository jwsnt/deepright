package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.block.BlockService;
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

@Slf4j
@Getter
@Setter
public class BlockSubmitAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-block";

    protected BlockService blockService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        this.blockService.submit(workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildQuery(workflowConfig, workTask));
    }

    protected String buildQuery(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return workTask.getQuery();
    }

    @ConditionalOnProperty(name = "block.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected BlockService blockService;

        @Bean(BlockSubmitAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = BlockSubmitAssistant.WORKFLOW_NAME)
        public BlockSubmitAssistant blockSubmitAssistant() throws Exception {
            BlockSubmitAssistant blockSubmitAssistant = new BlockSubmitAssistant();
            BeanUtils.copyProperties(this, blockSubmitAssistant);
            log.info("BlockSubmitAssistant inited");
            return blockSubmitAssistant;
        }
    }
}

