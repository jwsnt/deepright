package ai.open.right.workflow.mcp.config.impl;

import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.mcp.config.McpConfigInit;
import ai.open.right.workflow.mcp.config.McpConfigService;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.io.BufferedInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class McpConfigServiceImpl implements McpConfigService {

    protected PlaceholderResolver placeholderResolver;

    protected List<McpConfigInit> mcpConfigInit;

    protected ResourceService resourceService;

    // MCP配置文件路径（URI）
    protected String uri;

    @PostConstruct
    public void init() throws Exception {
        if (!StringUtils.isEmpty(this.uri) && !CollectionUtils.isEmpty(this.mcpConfigInit)) {
            if (log.isInfoEnabled()) {
                log.info("Reading mcp config file={}", this.uri);
            }
            Map<String, Object> config = this.config(this.uri);
            // 分发配置
            for (McpConfigInit init : this.mcpConfigInit) {
                init.init(config);
            }
        }
    }

    @Override
    // 读取配置文件的Config
    public Map<String, Object> config(String uri) throws Exception {
        try (InputStream input = new BufferedInputStream(this.resourceService.url(uri).openStream())) {
            String config = IOUtils.toString(input, StandardCharsets.UTF_8);
            Assert.hasText(config, "Mcp config can not be empty");
            config = this.placeholderResolver.replace(config);
            if (log.isInfoEnabled()) {
                log.info("Reading mcp config={}", config);
            }
            return JsonUtils.read(config, Map.class);
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Autowired
        protected List<McpConfigInit> mcpConfigInit;

        @Autowired
        protected ResourceService resourceService;

        @Value("${mcp.uri:}")
        // MCP配置文件路径（URI）
        protected String uri;

        @Bean
        @ConditionalOnMissingBean(value = McpConfigService.class)
        public McpConfigService mcpConfigService() throws Exception {
            McpConfigServiceImpl mcpConfigService = new McpConfigServiceImpl();
            BeanUtils.copyProperties(this, mcpConfigService);
            if (log.isDebugEnabled()) {
                log.debug("McpConfigServiceImpl inited, uri={}", mcpConfigService.getUri());
            }
            return mcpConfigService;
        }
    }
}
