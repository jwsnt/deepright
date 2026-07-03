package ai.deepright.workflow.assistant;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.memory.MemoryService;
import ai.deepright.memory.impl.DefMemoryService;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.PassageAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

import java.time.ZoneId;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
public class HistoryCloseAssistant extends PassageAssistant {

    public static final String LANG_KEY_ASSISTANT_CLOSE_HINT = "assistant.close.hint";

    protected MemoryService memoryService;

    protected Integer offset;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        if (FeatureFlag.isDaemon(workTask)) {
            // 后台任务标记不关闭通道，@See CliTaskFunction && KnowledgeService
            this.chainOr2Endpoint(workflowConfig, workTask, workTask.getQuery());
        } else {
            this.commit(workflowConfig, workTask);
            super.execute(workflowConfig, workTask);
        }
    }

    @Override
    protected String buildCloseContent(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            return StringUtils.repeat(System.lineSeparator(), 2) + XmlResourceLang.get(HistoryCloseAssistant.LANG_KEY_ASSISTANT_CLOSE_HINT).replace("#time", TimeUnit.SECONDS.convert(System.currentTimeMillis() - workTask.getCreated(), java.util.concurrent.TimeUnit.MILLISECONDS) + "'s");
        } else {
            return "";
        }
    }

    protected ZoneId buildZoneId(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        String timezone = FeatureUtils.buildTimezone(workTask);
        return StringUtils.isBlank(timezone) ? ZoneId.systemDefault() : ZoneId.of(timezone);
    }

    // 关闭前提交记忆
    protected void commit(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        this.memoryService.commit(workTask);
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
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
