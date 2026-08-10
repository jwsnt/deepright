package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class RagSchema extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_schema";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag schema start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildSchema(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected String buildSchema(RagConfig ragConfig, RagData ragData) throws Exception {
        Object schema = MapUtils.getObject(ragData.getConfig().getAdditional(), ProviderRequestService.KEY_RESPONSE_SCHEMA);
        if (schema != null) {
            StringBuffer buffer = new StringBuffer();
            buffer.append(System.lineSeparator()).append("```").append("JSON SCHEMA").append(System.lineSeparator());
            buffer.append(JsonUtils.write(schema));
            buffer.append(System.lineSeparator()).append("```").append(System.lineSeparator());
            return buffer.toString();
        } else {
            return "";
        }
    }

    @ConditionalOnProperty(name = "schema.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagSchema.RAG_KEY)
        @ConditionalOnMissingBean(name = RagSchema.RAG_KEY)
        public RagSchema ragSchema() throws Exception {
            RagSchema ragSchema = new RagSchema();
            BeanUtils.copyProperties(this, ragSchema);
            log.info("RagSchema inited, timeout4Condition={}", ragSchema.getTimeout4Condition());
            return ragSchema;
        }
    }
}


