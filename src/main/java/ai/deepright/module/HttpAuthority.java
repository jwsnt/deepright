package ai.deepright.module;

import static org.springframework.util.ObjectUtils.isEmpty;

import static org.springframework.util.StringUtils.hasText;




import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.deepright.auth.AuthService;
import ai.deepright.feature.FeatureUtils;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class HttpAuthority implements TokenMapping {

    public static final TokenEntry TOKEN = TokenEntry.builder().build();

    public static final String NAME = "token.http";

    protected AuthService authService;

    @Override
    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
        // TOKEN为空或cli@pub sub get task时不转写Provider和Token
        if (StringUtils.isEmpty(token) || StringUtils.equalsIgnoreCase("cli", workTask.getBiz())) {
            return HttpAuthority.TOKEN;
        }
        String provider = FeatureUtils.buildTargetProvider(workTask);
        WorkflowException.check(!hasText(provider), "The provider can not be empty", ProtocolCode.C400);
        if (this.authService.support(provider)) {
            // 验证
            this.authService.auth(workTask, provider, token);
        } else {
            // 其他服务商检查Token
            WorkflowException.check(!hasText(token), "The token can not be empty", ProtocolCode.C400);
            workTask.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, StringUtils.trim(token));
        }
        return HttpAuthority.TOKEN;
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected AuthService authService;

        @Bean(name = HttpAuthority.NAME)
        @ConditionalOnMissingBean(name = HttpAuthority.NAME)
        public HttpAuthority httpAuthority() throws Exception {
            HttpAuthority httpAuthority = new HttpAuthority();
            BeanUtils.copyProperties(this, httpAuthority);
            log.info("HttpAuthority inited");
            return httpAuthority;
        }
    }
}
