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
// 使用原始Query增强内容
public class RagOriginalQuery extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_query_original";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag original query start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), ragData.getQuery().getOriginal());
        return new RagAtOnce(ragConfig);
    }

    @ConditionalOnProperty(name = "query.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagOriginalQuery.RAG_KEY)
        @ConditionalOnMissingBean(name = RagOriginalQuery.RAG_KEY)
        public RagOriginalQuery ragOriginalQuery() throws Exception {
            RagOriginalQuery ragOriginalQuery = new RagOriginalQuery();
            BeanUtils.copyProperties(this, ragOriginalQuery);
            log.info("RagOriginalQuery inited, timeout4Condition={}", ragOriginalQuery.getTimeout4Condition());
            return ragOriginalQuery;
        }
    }
}
