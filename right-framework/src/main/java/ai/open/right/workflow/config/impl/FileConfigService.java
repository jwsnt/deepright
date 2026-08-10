package ai.open.right.workflow.config.impl;

import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.Config;
import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.config.ConfigService;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.io.File;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.Callable;

@Slf4j
@Setter
@Getter
public class FileConfigService implements ConfigService {

    public static final String NAME = "config.file";

    private final Cache<String, Config> configCache = CacheBuilder.newBuilder().maximumSize(Integer.MAX_VALUE).build();

    protected PlaceholderResolver placeholderResolver;

    protected ResourceService resourceService;

    // 思考链（Workflow）配置文件的后缀名
    protected String suffix;

    // 思考链（Workflow）配置文件的查找路径
    protected String path;

    @PostConstruct
    public void init() {
        this.path = this.path.endsWith(File.separator) ? this.path : this.path + File.separator;
        if (log.isInfoEnabled()) {
            log.info("Reading Workflow config path={}", this.path);
        }
    }

    public Config get(ConfigSearch configSearch) throws Exception {
        Config config = this.search(configSearch);
        Assert.notNull(config, "Can not find matched config, please check config: " + configSearch);
        return config;
    }

    public Config search(ConfigSearch configSearch) throws Exception {
        ConfigSearch.ConfigSearchChecker.check(configSearch);
        String key = this.path + configSearch.getBiz() + this.suffix;
        if (log.isDebugEnabled()) {
            log.debug("Loading config={}", key);
        }
        // 从缓存读取
        return this.configCache.get(key, new ConfigCallable(this.placeholderResolver, this.resourceService, key, configSearch.getBiz()));
    }

    public static class ConfigCallable implements Callable<Config> {

        // 替换占位符
        protected final PlaceholderResolver placeholderResolver;

        protected final ResourceService resourceService;

        protected final String key;

        protected final String biz;

        public ConfigCallable(PlaceholderResolver placeholderResolver, ResourceService resourceService, String key, String biz) {
            this.placeholderResolver = placeholderResolver;
            this.resourceService = resourceService;
            this.key = key;
            this.biz = biz;
        }

        @Override
        public Config call() throws Exception {
            try (InputStream input = new BufferedInputStream(this.resourceService.url(this.key).openStream())) {
                // 读取，替换占位符并转换
                Config config = new Config(this.biz, JsonUtils.read(this.placeholderResolver.replace(IOUtils.toString(input, StandardCharsets.UTF_8)), Map.class));
                Config.ConfigChecker.check(config);
                return config;
            }
        }
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Autowired
        protected ResourceService resourceService;

        @Value("${config.file.suffix:.json}")
        // 思考链（Workflow）配置文件的后缀名
        protected String suffix;

        @Value("${config.file.path:classpath:config/}")
        // 思考链（Workflow）配置文件的查找路径
        protected String path;

        @Bean(name = FileConfigService.NAME)
        @ConditionalOnMissingBean(name = FileConfigService.NAME)
        public ConfigService configService() throws Exception {
            FileConfigService fileConfigService = new FileConfigService();
            BeanUtils.copyProperties(this, fileConfigService);
            log.info("FileConfigService inited, path={},suffix={}", fileConfigService.getPath(), fileConfigService.getSuffix());
            return fileConfigService;
        }
    }
}
