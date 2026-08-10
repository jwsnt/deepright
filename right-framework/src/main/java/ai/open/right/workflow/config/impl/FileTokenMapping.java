package ai.open.right.workflow.config.impl;

import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.TokenEntry;
import ai.open.right.workflow.config.TokenMapping;
import ai.open.right.workflow.flow.WorkflowTask;
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
import java.util.Map;

@Setter
@Getter
@Slf4j
public class FileTokenMapping implements TokenMapping {

    public static final String KEY_WORKFLOW = "workflow";

    public static final String KEY_BIZ = "biz";

    public static final String NAME = "token.file";

    protected Map<String, Map<String, String>> mapping;

    protected PlaceholderResolver placeholderResolver;

    protected ResourceService resourceService;

    // 开启Token与Biz/Workflow映射的配置文件路径
    protected String path;

    @PostConstruct
    public void init() throws Exception {
        if (!StringUtils.isEmpty(this.path)) {
            try (InputStream input = new BufferedInputStream(this.resourceService.url(this.path).openStream())) {
                // 加载，替换，转换
                this.mapping = JsonUtils.read(this.placeholderResolver.replace(IOUtils.toString(input, StandardCharsets.UTF_8)), Map.class);
                for (String key : this.mapping.keySet()) {
                    Map<String, String> entry = this.mapping.get(key);
                    String workflow = entry.get(FileTokenMapping.KEY_WORKFLOW);
                    String biz = entry.get(FileTokenMapping.KEY_BIZ);
                    Assert.hasText(workflow, "Token's workflow can not be empty: " + key);
                    Assert.hasText(biz, "Token's biz can not be empty: " + key);
                }
            }
            log.info("Reading token={}", this.mapping);
        }
    }

    public TokenEntry entry(WorkflowTask workTask, String token) throws Exception {
        // 没有配置
        if (CollectionUtils.isEmpty(this.mapping)) {
            return null;
        }
        Map<String, String> mapping = this.mapping.get(StringUtils.trim(token));
        if (!CollectionUtils.isEmpty(mapping)) {
            String workflow = mapping.get(FileTokenMapping.KEY_WORKFLOW);
            String biz = mapping.get(FileTokenMapping.KEY_BIZ);
            if (log.isInfoEnabled()) {
                log.info("Loading token {}-{}-{}", token, biz, workflow);
            }
            TokenEntry entry = TokenEntry.builder()
                    .workflow(workflow)
                    .biz(biz)
                    .build();
            if (log.isDebugEnabled()) {
                log.debug("Token entry={}", entry);
            }
            return entry;
        } else {
            // 没有配置
            return null;
        }
    }

    @ConditionalOnProperty(name = "token.file.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Autowired
        protected ResourceService resourceService;

        @Value("${token.file.path:}")
        // 开启Token与Biz/Workflow映射的配置文件路径
        protected String path;

        @Bean(name = FileTokenMapping.NAME)
        @ConditionalOnMissingBean(name = FileTokenMapping.NAME)
        public TokenMapping tokenMapping() throws Exception {
            FileTokenMapping fileTokenMapping = new FileTokenMapping();
            BeanUtils.copyProperties(this, fileTokenMapping);
            log.info("FileTokenMapping inited, path={}", fileTokenMapping.getPath());
            return fileTokenMapping;
        }
    }
}
