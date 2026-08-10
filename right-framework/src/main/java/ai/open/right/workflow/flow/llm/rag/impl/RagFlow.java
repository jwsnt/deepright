package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
// 使用思考链（Workflow）增强内容
public class RagFlow extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_flow";

    // Rag Flow整体超时
    protected Integer timeout4Llm;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        return super.allowed(ragConfig, ragData) && ragConfig.hasRagOrchestrator() && ragConfig.getRagOrchestrator().hasBefore();
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag flow start");
        }
        return new RagFlow.FlowFuture(ragConfig, ragData, ragConfig.getTimeout(this.timeout4Llm));
    }

    public class FlowFuture extends RagAtOnce {

        protected final Integer timeout4llm;

        protected final RagData ragData;

        public FlowFuture(RagConfig ragConfig, RagData ragData, Integer timeout4llm) {
            super(ragConfig);
            this.ragData = ragData;
            this.timeout4llm = timeout4llm;
        }

        public void run() throws Exception {
            SyncConfig syncBefore = SyncConfig.builder().workflow(this.getRagConfig().getRagOrchestrator().getBefore()).reQuery(this.ragData.getQuery().getQuery()).workTask(this.ragData.getQuery()).timeout(this.timeout4llm).build();
            String originalResponse = SyncWorkflowTask.exeWorkflow(RagFlow.this.notifierService, syncBefore).get();
            if (log.isDebugEnabled()) {
                log.debug("The original response={}", originalResponse);
            }
            Assert.hasText(originalResponse, "The original response can not be empty");
            if (this.getRagConfig().getRagOrchestrator().hasAfter()) {
                // 如果需要二次清洗
                SyncConfig syncClearResponse = SyncConfig.builder().workflow(this.getRagConfig().getRagOrchestrator().getAfter()).workTask(this.ragData.getQuery()).reQuery(originalResponse).timeout(this.timeout4llm).build();
                originalResponse = SyncWorkflowTask.exeWorkflow(RagFlow.this.notifierService, syncClearResponse).get();
                if (log.isDebugEnabled()) {
                    log.debug("The cleaned response={}", originalResponse);
                }
            }
            Assert.hasText(originalResponse, "The cleaned response can not be empty");
            RagService.updatePrompt(this.getRagConfig(), this.ragData, this.getRagConfig().getReplace(), originalResponse);
        }
    }

    @ConditionalOnProperty(name = "flow.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Value("${flow.timeout.llm:1800000}")
        // Rag Flow整体超时
        protected Integer timeout4Llm;

        @Bean(RagFlow.RAG_KEY)
        @ConditionalOnMissingBean(name = RagFlow.RAG_KEY)
        public RagFlow ragFlow() throws Exception {
            RagFlow ragFlow = new RagFlow();
            BeanUtils.copyProperties(this, ragFlow);
            log.info("RagFlow inited: timeout4Llm={},timeout4Condition={}", ragFlow.getTimeout4Llm(), ragFlow.getTimeout4Condition());
            return ragFlow;
        }
    }
}
