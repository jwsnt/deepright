package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

public interface ProviderRequestRewriter {

    public void rewrite(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception;

    public static class BaseRequestRewriter implements ProviderRequestRewriter {

        @Override
        public void rewrite(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

        }
    }
    @Configuration
    @Setter
    @Getter
    @Slf4j
    public static class InitConfig {

        @Bean
        @ConditionalOnMissingBean(value = ProviderRequestRewriter.class)
        public BaseRequestRewriter requestRewriter() throws Exception {
            BaseRequestRewriter baseRequestRewriter = new BaseRequestRewriter();
            BeanUtils.copyProperties(this, baseRequestRewriter);
            log.info("BaseRequestRewriter inited");
            return baseRequestRewriter;
        }
    }
}
