package ai.open.right.workflow.ratelimit.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.ratelimit.RateLimitService;
import com.google.common.util.concurrent.RateLimiter;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
public class GuavaRateLimitService implements RateLimitService {

    // 限流（@limit）
    protected RateLimiter rateLimiter;

    // 最大同时执行并发数
    protected Integer limit;

    @PostConstruct
    public void init() {
        this.rateLimiter = RateLimiter.create(this.limit);
    }

    // 检查是否限流
    public void checkLimit(WorkflowTask workTask) throws Exception {
        if (!this.rateLimiter.tryAcquire()) {
            throw new WorkflowException("Too many requests: You have been rate limited", ProtocolCode.C430);
        }
    }

    @ConditionalOnProperty(name = "rateLimit.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Value("${ratelimit:25}")
        // 最大同时执行并发数
        protected Integer limit;

        @Bean
        @ConditionalOnMissingBean(value = RateLimitService.class)
        public RateLimitService rateLimitService() throws Exception {
            GuavaRateLimitService rateLimitService = new GuavaRateLimitService();
            BeanUtils.copyProperties(this, rateLimitService);
            log.info("GuavaRateLimitService inited: limit={}", rateLimitService.getLimit());
            return rateLimitService;
        }
    }
}
