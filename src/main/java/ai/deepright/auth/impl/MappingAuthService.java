package ai.deepright.auth.impl;

import ai.deepright.feature.FeatureField;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.deepright.auth.AuthService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Getter
@Setter
@Slf4j
public class MappingAuthService implements AuthService {

    // 输入源
    protected String sourceProvider;

    // 切换源
    protected String targetProvider;

    protected String sourceToken;

    protected String targetToken;

    @Override
    public void auth(WorkflowTask workTask, String provider, String token) throws Exception {
        this.check(workTask, provider, token);
        this.replace(workTask, provider, token);
    }

    @Override
    public Boolean support(String provider) throws Exception {
        return StringUtils.equalsIgnoreCase(provider, this.sourceProvider);
    }

    protected void replace(WorkflowTask workTask, String provider, String token) throws Exception {
        workTask.setProviderAndToken(this.targetProvider, this.targetToken);
        workTask.putMetadata(FeatureField.KEY_PROVIDER, provider);

    }

    protected void check(WorkflowTask workTask, String provider, String token) throws Exception {
        // 区分大小写
        if (!StringUtils.equals(token, this.sourceToken)) {
            throw new WorkflowException("The token is invalid", ProtocolCode.C401).needExposure();
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Value("${auth.source.provider:deepright}")
        protected String sourceProvider;

        @Value("${auth.target.provider}")
        protected String targetProvider;

        @Value("${auth.source.token}")
        protected String sourceToken;

        @Value("${auth.target.token}")
        protected String targetToken;

        @Bean(name = AuthService.NAME)
        @ConditionalOnMissingBean(name = AuthService.NAME)
        public AuthService authService() throws Exception {
            MappingAuthService authService = new MappingAuthService();
            BeanUtils.copyProperties(this, authService);
            log.info("MappingAuthService inited");
            return authService;
        }
    }
}
