package ai.deepright.workflow.assistant;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.memory.MemoryService;
import ai.deepright.memory.impl.DefMemoryService;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.PassageAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class HistoryCloseAssistant extends PassageAssistant {

    protected MemoryService memoryService;

    protected Integer offset;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 测试请求只用于连通性探测，绝不能提交或更新记忆。
        if (!FeatureFlag.isTest(workTask)) {
            // Cron、Task、Daemon、普通会话所有场景均触发commit
            this.commit(workflowConfig, workTask);
        }
        if (FeatureFlag.isDaemon(workTask)) {
            // 后台任务标记不关闭通道，@See CliTaskFunction && KnowledgeService
            this.chainOr2Endpoint(workflowConfig, workTask, workTask.getQuery());
        } else {
            super.execute(workflowConfig, workTask);
        }
    }

    @Override
    protected String buildCloseContent(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return "";
    }

    // 关闭前提交记忆
    protected void commit(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        this.memoryService.commit(workTask);
    }

    @Configuration
    @Setter
    @Getter
    public static class DefaultInitConfig extends InitConfig {

        @Autowired
        @Qualifier(DefMemoryService.NAME)
        protected MemoryService memoryService;

        @Autowired
        protected WorkflowQueue workflowQueue;

        @Value("${llm.recallOffset:}")
        protected Integer offset;

        @Override
        @Bean(PassageAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = PassageAssistant.WORKFLOW_NAME)
        public HistoryCloseAssistant passageAssistant() throws Exception {
            HistoryCloseAssistant historyCloseAssistant = new HistoryCloseAssistant();
            BeanUtils.copyProperties(this, historyCloseAssistant);
            log.info("HistoryCloseAssistant inited");
            return historyCloseAssistant;
        }
    }
}
