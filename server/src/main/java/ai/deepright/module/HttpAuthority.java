package ai.deepright.module;

import ai.deepright.auth.AuthService;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import com.google.common.collect.ImmutableMap;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
public class HttpAuthority implements TokenMapping {

    public static final TokenEntry TOKEN = TokenEntry.builder().build();

    public static final String LANG_KEY_MESSAGE = "auth.message";

    public static final String NAME = "token.http";

    public static final Integer NEW_TAB = 931;

    public static final Integer IFRAME = 932;

    protected AuthService authService;

    protected String response;

    protected Integer code;

    protected String url;

    @PostConstruct
    public void init() throws Exception {
        this.response = JsonUtils.write(ImmutableMap.of("url", this.url, "message", XmlResourceLang.get(HttpAuthority.LANG_KEY_MESSAGE)));
    }

    @Override
    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
        this.checkDevice(workTask, token);
        return this.buildToken(workTask, token);
    }

    protected TokenEntry buildToken(WorkflowTask workTask, String token) throws Exception {
        // TOKEN为空或cli@pub sub get task时不转写Provider和Token
        if (StringUtils.isEmpty(token) || StringUtils.equalsIgnoreCase("cli", workTask.getBiz())) {
            return HttpAuthority.TOKEN;
        }
        String provider = FeatureUtils.buildTargetProvider(workTask);
        WorkflowException.checkCondition(StringUtils.isEmpty(provider), "The provider can not be empty");
        if (this.authService.support(provider)) {
            // 验证
            this.authService.auth(workTask, provider, token);
        } else {
            // 其他服务商检查Token
            WorkflowException.checkCondition(StringUtils.isEmpty(token), "The token can not be empty");
            workTask.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, StringUtils.trim(token));
        }
        return HttpAuthority.TOKEN;
    }

    // 由子类覆盖
    protected Boolean allowedDevice(WorkflowTask workTask, String token) throws Exception {
        return true;
    }

    // 检查Device
    protected void checkDevice(WorkflowTask workTask, String token) throws Exception {
        WorkflowException.checkCondition(!this.allowedDevice(workTask, token), true, this.response, this.code);
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected AuthService authService;

        @Value("${auth.code:}")
        protected Integer code = HttpAuthority.IFRAME;

        @Value("${auth.url:}")
        protected String url;

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
