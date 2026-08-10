package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.context.UserContext;
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
// 使用用户信息（UserContext）增强内容
public class RagUser extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_user";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag user start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildUserContext(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected Object buildUserContext(RagConfig ragConfig, RagData ragData) throws Exception {
        UserContext userContext = ragData.getQuery().getUserContext();
        if (ragConfig.isMode(RagConfig.MODE_JSON)) {
            return userContext;
        }
        LLMUserPrompts llmUserPrompts = LLMUserPrompts.builder()
                .language(userContext.getLanguage())
                .region(userContext.getRegion())
                .system(userContext.getSystem())
                .device(userContext.getDevice())
                .brand(userContext.getBrand())
                .model(userContext.getModel())
                .build();
        if (log.isDebugEnabled()) {
            log.debug("The rag user={}", llmUserPrompts);
        }
        return llmUserPrompts;
    }

    @Getter
    @Builder
    @JacksonXmlRootElement(localName = "User")
    @JsonInclude(JsonInclude.Include.NON_NULL)
    public static class LLMUserPrompts {

        protected String language;

        protected String system;

        protected String device;

        protected String region;

        protected String brand;

        protected String model;
    }

    @ConditionalOnProperty(name = "user.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagUser.RAG_KEY)
        @ConditionalOnMissingBean(name = RagUser.RAG_KEY)
        public RagUser ragUser() throws Exception {
            RagUser ragUser = new RagUser();
            BeanUtils.copyProperties(this, ragUser);
            log.info("RagUser inited, timeout4Condition={}", ragUser.getTimeout4Condition());
            return ragUser;
        }
    }
}
