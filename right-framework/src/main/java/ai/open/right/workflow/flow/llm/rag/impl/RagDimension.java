package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlRootElement;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class RagDimension extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_dimension";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag dimension start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildDimension(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected Object buildDimension(RagConfig ragConfig, RagData ragData) throws Exception {
        return LLMDimension.builder()
                .device(ragData.getQuery().getUserContext().getDevice())
                .workflow(ragData.getQuery().getWorkflow())
                .chat(ragData.getQuery().getChat())
                .biz(ragData.getQuery().getBiz())
                .build();
    }

    @Getter
    @Builder
    @JacksonXmlRootElement(localName = "Dimension")
    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class LLMDimension {

        protected String workflow;

        protected String device;

        protected String chat;

        protected String biz;
    }

    @ConditionalOnProperty(name = "dimension.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagDimension.RAG_KEY)
        @ConditionalOnMissingBean(name = RagDimension.RAG_KEY)
        public RagDimension ragDimension() throws Exception {
            RagDimension ragDimension = new RagDimension();
            BeanUtils.copyProperties(this, ragDimension);
            log.info("RagDimension inited, timeout4Condition={}", ragDimension.getTimeout4Condition());
            return ragDimension;
        }
    }
}
