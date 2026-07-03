package ai.deepright.safety.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.deepright.safety.SafetyService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
public class SafetyServiceImpl implements SafetyService {

    public static final String NAME = "safety_service";

    @Override
    public String safety(WorkflowTask workTask) throws Exception {
        // 默认空，子类实现
        return "";
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    public static class InitConfig {

        @Bean(SafetyServiceImpl.NAME)
        @ConditionalOnMissingBean(name = SafetyServiceImpl.NAME)
        public SafetyServiceImpl safetyService() throws Exception {
            log.info("SafetyServiceImpl inited");
            return new SafetyServiceImpl();
        }
    }
}
