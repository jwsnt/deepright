package ai.open.right.workflow.flow.llm.provider.reason.impl;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class ProviderReasonImpl implements ProviderReason {

    @Override
    public String reason(ProviderRequest request, String message, Boolean finished, Integer index) throws Exception {
        return message;
    }
    @Setter
    @Getter
    @Configuration
    public static class InitConfig {

        @Bean
        @ConditionalOnMissingBean(value = ProviderReason.class)
        public ProviderReason providerReason() throws Exception {
            ProviderReason providerReason = new ProviderReasonImpl();
            BeanUtils.copyProperties(this, providerReason);
            log.info("ProviderReason inited");
            return providerReason;
        }
    }
}
