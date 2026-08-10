package ai.open.right.workflow.config.impl;

import ai.open.right.utils.SplitUtils;
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
@Setter
@Getter
public class DyTokenMapping implements TokenMapping {

    public static final String NAME = "token.dy";

    @Override
    // 将Token（biz@workflow）切分为2部分
    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
        String[] part = SplitUtils.split(token);
        TokenEntry entry = TokenEntry.builder().workflow(part[1]).biz(part[0]).build();
        if (log.isDebugEnabled()) {
            log.debug("Token entry={}", entry);
        }
        return entry;
    }

    @ConditionalOnProperty(name = "token.dynamic.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Bean(name = DyTokenMapping.NAME)
        @ConditionalOnMissingBean(name = DyTokenMapping.NAME)
        public TokenMapping tokenMapping() throws Exception {
            DyTokenMapping dyTokenMapping = new DyTokenMapping();
            log.info("DyTokenMapping inited");
            return dyTokenMapping;
        }
    }
}
