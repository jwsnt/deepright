package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class NoneTokenMapping implements TokenMapping {

    public static final String NAME = "token.none";

    public static TokenEntry NONE = TokenEntry.builder().build();

    @Override
    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
        return NoneTokenMapping.NONE;
    }

    @ConditionalOnProperty(name = "token.none.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Bean(name = NoneTokenMapping.NAME)
        @ConditionalOnMissingBean(name = NoneTokenMapping.NAME)
        public TokenMapping tokenMapping() throws Exception {
            NoneTokenMapping noneTokenMapping = new NoneTokenMapping();
            log.info("NoneTokenMapping inited");
            return noneTokenMapping;
        }
    }
}
