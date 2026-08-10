package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.concurrent.ThreadLocalRandom;

@Slf4j
@Getter
@Setter
public class ProviderToken {

    public static final String NAME = "ProviderToken";

    // Token可能是以逗号分割的字符串（也可能不分割就一个），若分割后大于等于2个则随机取数组里的任一个返回。
    public String select(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery, String token) throws Exception {
        if (StringUtils.isEmpty(token)) {
            return "";
        }
        String[] parts = StringUtils.split(token, ",");
        if (parts.length > 1) {
            int index = ThreadLocalRandom.current().nextInt(0, parts.length);
            if (log.isDebugEnabled()) {
                log.debug("The provider token index={}", index);
            }
            return StringUtils.trim(parts[index]);
        } else {
            return StringUtils.trim(parts[0]);
        }
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Bean(name = ProviderToken.NAME)
        @ConditionalOnMissingBean(name = ProviderToken.NAME)
        public ProviderToken providerToken() throws Exception {
            ProviderToken providerToken = new ProviderToken();
            BeanUtils.copyProperties(this, providerToken);
            log.info("ProviderToken inited.");
            return providerToken;
        }
    }
}
