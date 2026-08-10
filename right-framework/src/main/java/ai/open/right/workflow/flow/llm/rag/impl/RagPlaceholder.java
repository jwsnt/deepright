package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.resouce.PlaceholderResolver;
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
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class RagPlaceholder extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_placeholder";

    protected PlaceholderResolver placeholderResolver;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        return super.allowed(ragConfig, ragData);
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag placeholder start");
        }
        String prompt = this.placeholderResolver.replace(ragData.getPrompt());
        if (log.isDebugEnabled()) {
            log.debug("Rag prompt{}", prompt);
        }
        ragData.setPrompt(prompt);
        return new RagAtOnce(ragConfig);
    }

    @ConditionalOnProperty(name = "placeholder.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Bean(RagPlaceholder.RAG_KEY)
        @ConditionalOnMissingBean(name = RagPlaceholder.RAG_KEY)
        public RagPlaceholder ragPlaceholder() throws Exception {
            RagPlaceholder ragPlaceholder = new RagPlaceholder();
            BeanUtils.copyProperties(this, ragPlaceholder);
            log.info("RagPlaceholder inited, timeout4Condition={}", ragPlaceholder.getTimeout4Condition());
            return ragPlaceholder;
        }
    }
}

