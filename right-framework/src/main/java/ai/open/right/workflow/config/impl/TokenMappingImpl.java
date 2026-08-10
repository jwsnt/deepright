package ai.open.right.workflow.config.impl;


import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.WorkflowTask;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class TokenMappingImpl implements TokenMapping {

    public static final String NAME = "TokenMappingImpl";

    protected Map<String, TokenMapping> tokenMapping;

    protected TokenMapping firstMapping;

    // 动态话拆解Token为Biz/Workflow
    protected TokenMapping defMapping;

    // Token配置的加载方式，默认为DyTokenMapping（token.dy）
    protected String instance;

    @PostConstruct
    public void init() throws Exception {
        if (!MapUtils.isEmpty(this.tokenMapping)) {
            this.firstMapping = this.tokenMapping.values().stream().findFirst().orElse(null);
        }
    }

    @Override
    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
        TokenMapping tokenMapping = this.tokenMapping.get(StringUtils.trim(this.instance));
        TokenEntry tokenEntry = tokenMapping != null ? tokenMapping.entry(workTask, token) : this.buildDefault(workTask, token);
        Assert.notNull(tokenEntry, "The token entry can not be empty: " + this.instance);
        return tokenEntry;
    }

    protected TokenEntry buildDefault(WorkflowTask workTask, String token) throws Exception {
        TokenMapping finalMapping = this.defMapping != null ? this.defMapping : this.firstMapping;
        Assert.notNull(finalMapping, "The final mapping can not be empty");
        return finalMapping.entry(workTask, token);
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Map<String, TokenMapping> tokenMapping;

        @Autowired(required = false)
        @Qualifier("token.dy")
        // 动态话拆解Token为Biz/Workflow
        protected TokenMapping defMapping;

        @Value("${token.def:}")
        // Token配置的加载方式，默认为DyTokenMapping（token.dy）
        protected String instance;

        @Bean(name = TokenMappingImpl.NAME)
        @ConditionalOnMissingBean(name = TokenMappingImpl.NAME)
        public TokenMappingImpl tokenManager() throws Exception {
            TokenMappingImpl tokenManager = new TokenMappingImpl();
            BeanUtils.copyProperties(this, tokenManager);
            log.info("TokenManager inited: instance={}", tokenManager.getInstance());
            return tokenManager;
        }
    }
}
