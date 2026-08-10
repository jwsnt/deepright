package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlElementWrapper;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlProperty;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlRootElement;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlText;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class RagEnv extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_env";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag env start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildEnv(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected Object buildEnv(RagConfig ragConfig, RagData ragData) throws Exception {
        if (ragConfig.isMode(RagConfig.MODE_JSON)) {
            return ragConfig.buildEnvironment();
        }
        LLMEnvPrompts envPrompts = new LLMEnvPrompts();
        Map<String, String> environment = ragConfig.buildEnvironment();
        for (String key : environment.keySet()) {
            LLMInputPrompts input = LLMInputPrompts.builder()
                    .val(environment.get(key))
                    .key(key)
                    .build();
            envPrompts.add(input);
        }
        if (log.isDebugEnabled()) {
            log.debug("The rag env={}", envPrompts);
        }
        return envPrompts;
    }


    @Getter
    @JacksonXmlRootElement(localName = "Env")
    public static class LLMEnvPrompts {

        @JacksonXmlElementWrapper(useWrapping = false)
        @JacksonXmlProperty(localName = "Input")
        protected List<LLMInputPrompts> input;

        public LLMEnvPrompts add(LLMInputPrompts input) {
            if (this.input == null) {
                this.input = new ArrayList<LLMInputPrompts>();
            }
            this.input.add(input);
            return this;
        }
    }

    @Getter
    @Builder
    @JacksonXmlRootElement(localName = "Input")
    public static class LLMInputPrompts {

        @JacksonXmlText
        protected Object val;

        @JacksonXmlProperty(isAttribute = true)
        protected String key;
    }

    @ConditionalOnProperty(name = "env.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagEnv.RAG_KEY)
        @ConditionalOnMissingBean(name = RagEnv.RAG_KEY)
        public RagEnv ragEnv() throws Exception {
            RagEnv ragEnv = new RagEnv();
            BeanUtils.copyProperties(this, ragEnv);
            log.info("RagEnv inited, timeout4Condition={}", ragEnv.getTimeout4Condition());
            return ragEnv;
        }
    }
}