package ai.open.right.workflow.config.impl;

import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.PromptSearch;
import ai.open.right.workflow.config.PromptService;
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
import org.springframework.util.StringUtils;

import java.io.BufferedInputStream;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Callable;

@Slf4j
@Setter
@Getter
public class FilePromptService implements PromptService {

    public static final String NAME = "prompt.file";

    // MaximumSize由使用者控制
    private final Cache<String, Prompt> promptCache = CacheBuilder.newBuilder().maximumSize(Integer.MAX_VALUE).build();

    protected PlaceholderResolver placeholderResolver;

    protected ResourceService resourceService;

    // System Prompt配置文件的后缀名
    protected String suffix = "";

    // System Prompt配置文件的查找路径
    protected String path;

    @PostConstruct
    public void init() throws Exception {
        if (StringUtils.hasText(this.path) && this.path.endsWith(File.separator)) {
            this.path = this.path.substring(0, this.path.length() - 1);
        }
        if (log.isInfoEnabled()) {
            log.info("Reading Workflow prompt path={}", this.path);
        }
    }

    @Override
    public Prompt get(PromptSearch promptSearch) throws Exception {
        Prompt prompt = this.search(promptSearch);
        Assert.notNull(prompt, "Can not find matched prompt, please check config: " + promptSearch);
        return prompt;
    }

    @Override
    public Prompt search(PromptSearch promptSearch) throws Exception {
        PromptSearch.PromptSearchChecker.check(promptSearch);
        String key = this.path + File.separator + promptSearch.getBiz() + File.separator + promptSearch.getPrompt() + this.suffix;
        if (log.isInfoEnabled()) {
            log.info("Loading prompt={}", key);
        }
        // 从缓存读取
        return this.promptCache.get(key, new PromptCallable(this.placeholderResolver, this.resourceService, promptSearch.getWorkTask().getWorkflow(), promptSearch.getPrompt(), promptSearch.getBiz(), key));
    }

    public static class PromptCallable implements Callable<Prompt> {

        protected final PlaceholderResolver placeholderResolver;

        protected final ResourceService resourceService;

        protected final String workflow;

        protected final String prompt;

        protected final String key;

        protected final String biz;

        public PromptCallable(PlaceholderResolver placeholderResolver, ResourceService resourceService, String workflow, String prompt, String biz, String key) {
            this.placeholderResolver = placeholderResolver;
            this.resourceService = resourceService;
            this.workflow = workflow;
            this.prompt = prompt;
            this.biz = biz;
            this.key = key;
        }

        @Override
        public Prompt call() throws Exception {
            try (InputStream input = new BufferedInputStream(this.resourceService.url(this.key).openStream())) {
                // 读取，替换占位符并转换
                Prompt prompt = new Prompt(this.biz, this.prompt, this.placeholderResolver.replace(IOUtils.toString(input, StandardCharsets.UTF_8)));
                if (log.isInfoEnabled()) {
                    log.info("Loading prompt from disk={}", this.key);
                }
                Prompt.PromptChecker.check(prompt);
                return prompt;
            } catch (IOException e) {
                if (log.isDebugEnabled()) {
                    log.debug(e.getMessage(), e);
                }
                // 对应__fun_call等内部Prompt
                return new Prompt(this.biz, this.workflow, "");
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

        @Value("${prompt.file.suffix:}")
        // System Prompt配置文件的后缀名
        protected String suffix = "";

        @Value("${prompt.file.path:classpath:config/}")
        // System Prompt配置文件的查找路径
        protected String path;

        @Bean(name = FilePromptService.NAME)
        @ConditionalOnMissingBean(name = FilePromptService.NAME)
        public PromptService promptService() throws Exception {
            FilePromptService filePromptService = new FilePromptService();
            BeanUtils.copyProperties(this, filePromptService);
            log.info("FilePromptService inited, path={},suffix={}", filePromptService.getPath(), filePromptService.getSuffix());
            return filePromptService;
        }
    }
}
