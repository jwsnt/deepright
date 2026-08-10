package ai.open.right.workflow.flow.assistant;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.Notifier;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
// 特殊通道
public class PassageAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-passage";

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 先检查Close + Discard
        // 后检查单独Discard
        if (workflowConfig.getClose()) {
            if (log.isDebugEnabled()) {
                log.debug("Passage will close the channel");
            }
            this.close(workflowConfig, workTask);
        } else if (!workflowConfig.getDiscard()) {
            // 指定响应码（Code）
            this.chainOr2Endpoint(workflowConfig, workTask, workTask.getQuery(), workflowConfig.getCode());
        }
    }

    protected String buildCloseContent(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return !workflowConfig.getDiscard() ? workTask.getQuery() : "";
    }

    // 主动关闭
    protected void close(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 使用Code=0主动关闭
        Segment segment = Segment.build(workTask, Segment.SegmentConfig.builder()
                .content(new StringBuffer(this.buildCloseContent(workflowConfig, workTask)))
                .notifier(Notifier.SOURCE)
                .code(ProtocolCode.C0)
                .build());
        super.notifierService.notify(segment, workTask);
    }

    @ConditionalOnProperty(name = "passage.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Bean(PassageAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = PassageAssistant.WORKFLOW_NAME)
        public PassageAssistant passageAssistant() throws Exception {
            PassageAssistant passageAssistant = new PassageAssistant();
            BeanUtils.copyProperties(this, passageAssistant);
            log.info("PassageAssistant inited");
            return passageAssistant;
        }
    }
}
