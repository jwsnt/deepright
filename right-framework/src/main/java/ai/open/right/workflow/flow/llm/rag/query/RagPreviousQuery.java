package ai.open.right.workflow.flow.llm.rag.query;

import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
// 使用上一个思考链（Workflow）的Query增强内容
public class RagPreviousQuery extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_query_previous";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag previous query start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), ragData.getQuery().getPrevious());
        return new RagAtOnce(ragConfig);
    }

    @ConditionalOnProperty(name = "query.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagPreviousQuery.RAG_KEY)
        @ConditionalOnMissingBean(name = RagPreviousQuery.RAG_KEY)
        public RagPreviousQuery ragPreviousQuery() throws Exception {
            RagPreviousQuery ragPreviousQuery = new RagPreviousQuery();
            BeanUtils.copyProperties(this, ragPreviousQuery);
            log.info("RagPreviousQuery inited, timeout4Condition={}", ragPreviousQuery.getTimeout4Condition());
            return ragPreviousQuery;
        }
    }
}
