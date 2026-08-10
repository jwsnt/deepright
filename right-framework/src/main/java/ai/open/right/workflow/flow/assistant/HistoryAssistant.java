package ai.open.right.workflow.flow.assistant;

import ai.open.right.protocol.Protocol;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryClearConfig;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.summary.SummaryPart;
import ai.open.right.workflow.flow.summary.SummaryService;
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

import java.util.List;

@Slf4j
@Setter
@Getter
// 记忆处理
public class HistoryAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-history";

    protected SummaryService summaryService;

    protected HistoryStore historyStore;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasHistory() && workflowConfig.getHistoryConfig().isValid(), "History config is invalid, please check config");
        if (workflowConfig.getHistoryConfig().hasSummary()) {
            if (log.isDebugEnabled()) {
                log.debug("History summary");
            }
            // 总结
            this.summarize(workflowConfig, workTask);
            return;
        }
        if (workflowConfig.getHistoryConfig().hasClear()) {
            if (log.isDebugEnabled()) {
                log.debug("History clear");
            }
            // 清理
            this.clear(workflowConfig, workTask);
            return;
        }
        if (workflowConfig.getHistoryConfig().hasFetch()) {
            if (log.isDebugEnabled()) {
                log.debug("History fetch");
            }
            // 获取当前场景的会话记录
            this.fetch(workflowConfig, workTask);
            return;
        }
    }

    protected void summarize(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        SummaryPart summaryPart = this.summaryService.summarize(workflowConfig.getHistoryConfig().getSummaryConfig(), workTask);
        if (summaryPart == null) {
            return;
        }
        this.chainOr2Endpoint(workflowConfig, workTask, Protocol.CHAT, this.buildSummarizeQuery(workflowConfig, workTask, summaryPart));
    }

    protected void fetch(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 先看配置，后读Query（不使用Trim防止破坏MarkDown）
        String scene = StringUtils.defaultIfEmpty(workflowConfig.getHistoryConfig().getFetchConfig().getScene(), StringUtils.defaultIfEmpty(workTask.getQuery(), null));
        List<History> histories = this.historyStore.restore(workTask, scene, workflowConfig.getHistoryConfig().getFetchConfig().getNums());
        this.chainOr2Endpoint(workflowConfig, workTask, Protocol.CHAT, this.buildFetchQuery(workflowConfig, workTask, histories));
    }

    protected void clear(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        HistoryClearConfig historyClearConfig = workflowConfig.getHistoryConfig().getClearConfig();
        // 按配置清理会话（可能为多个），如果没配置则默认清理上游思考链（Workflow）
        // 按配置清理时间偏移量（Offset），UpStream为BIZ@Workflow
        // 先读Query，后看UpStream
        String scene = StringUtils.defaultIfBlank(StringUtils.defaultIfEmpty(workTask.getQuery(), null), workTask.getUpstream());
        // 时间取负数
        // desc=true, 更早的（created ≤ 基准时间）清掉旧历史，保留新的一段
        this.historyStore.clear(workTask, historyClearConfig.getRepositories(scene), true, -historyClearConfig.getOffset(workTask.getCreated()));
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildClearQuery(workflowConfig, workTask));
    }

    protected String buildSummarizeQuery(WorkflowConfig workflowConfig, WorkflowTask workTask, SummaryPart summaryPart) throws Exception {
        return JsonUtils.write(summaryPart);
    }

    protected String buildFetchQuery(WorkflowConfig workflowConfig, WorkflowTask workTask, List<History> histories) throws Exception {
        return JsonUtils.write(histories);
    }

    protected String buildClearQuery(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return workTask.getQuery();
    }

    @ConditionalOnProperty(name = "history.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected SummaryService summaryService;

        @Autowired
        protected HistoryStore historyStore;

        @Bean(HistoryAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = HistoryAssistant.WORKFLOW_NAME)
        public HistoryAssistant historyAssistant() throws Exception {
            HistoryAssistant historyAssistant = new HistoryAssistant();
            BeanUtils.copyProperties(this, historyAssistant);
            log.info("HistoryAssistant inited");
            return historyAssistant;
        }
    }
}
