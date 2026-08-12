package ai.open.right.workflow.flow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.select.ChainSelectService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
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
// 动态选择下一个思考联（Workflow）
public class ChainSelectAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-chainSelector";

    protected ChainSelectService chainSelectService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasSelector(), "ChainSelect `selector` can not be empty, please check config");
        String chain = this.chainSelectService.selectChain(workflowConfig.getChainSelectConfig(), workTask);
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                .content(workTask.getQuery() != null ? new StringBuffer(workTask.getQuery()) : null)
                .metadata(workTask.getMetadata())
                .workflow(chain)
                .build();
        Segment segment = Segment.build(workTask, segmentConfig);
        this.notifierService.notify(segment, workTask, workTask);
    }

    @ConditionalOnProperty(name = "chainSelector.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected ChainSelectService chainSelectService;

        @Bean(ChainSelectAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ChainSelectAssistant.WORKFLOW_NAME)
        public ChainSelectAssistant competitionAssistant() throws Exception {
            ChainSelectAssistant chainSelectAssistant = new ChainSelectAssistant();
            BeanUtils.copyProperties(this, chainSelectAssistant);
            log.info("ChainSelectAssistant inited");
            return chainSelectAssistant;
        }
    }
}
