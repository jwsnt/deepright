package ai.open.right.trace.impl;

import ai.open.right.trace.TraceService;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.builder.ToStringBuilder;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.StringUtils;

import java.util.UUID;

@Slf4j
public class TraceServiceImpl implements TraceService {

    @Override
    public String getTrace(String trace) {
        // 不存在则使用UUID
        return StringUtils.hasText(trace) ? trace : UUID.randomUUID().toString();
    }

    @Configuration
    public static class InitConfig {

        @Bean
        @ConditionalOnMissingBean(value = TraceService.class)
        public TraceService traceService() throws Exception {
            TraceServiceImpl traceService = new TraceServiceImpl();
            log.info("TraceServiceImpl inited={}", ToStringBuilder.reflectionToString(traceService));
            return traceService;
        }
    }
}
