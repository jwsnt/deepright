package ai.open.right.workflow.config.impl;

import ai.open.right.workflow.config.Config;
import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.config.ConfigService;
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
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class ConfigServiceImpl implements ConfigService {

    // Bean Name
    public static final String NAME = "ConfigServiceImpl";

    protected Map<String, ConfigService> configService;

    // 思考链（Workflow）配置的加载方式，默认为FileConfigService（config.file）
    protected String instance;

    @Override
    public Config get(ConfigSearch configSearch) throws Exception {
        return this.instance().get(configSearch);
    }

    @Override
    public Config search(ConfigSearch configSearch) throws Exception {
        return this.instance().search(configSearch);
    }

    public ConfigService instance() {
        ConfigService configService = this.configService.get(StringUtils.trim(this.instance));
        Assert.notNull(configService, "ConfigService can not be empty: " + this.instance);
        return configService;
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Map<String, ConfigService> configService;

        @Value("${config:config.file}")
        // 思考链（Workflow）配置的加载方式，默认为FileConfigService（config.file）
        protected String instance;

        @Bean(name = ConfigServiceImpl.NAME)
        @ConditionalOnMissingBean(name = ConfigServiceImpl.NAME)
        public ConfigService configService() throws Exception {
            ConfigServiceImpl configService = new ConfigServiceImpl();
            BeanUtils.copyProperties(this, configService);
            log.info("ConfigServiceImpl inited: instance={}", configService.getInstance());
            return configService;
        }
    }
}