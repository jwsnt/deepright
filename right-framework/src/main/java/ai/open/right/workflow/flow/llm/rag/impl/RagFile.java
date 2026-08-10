package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.Callable;

@Slf4j
@Setter
@Getter
public class RagFile extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_file";

    // MaximumSize由使用者控制
    private final Cache<String, String> fileCache = CacheBuilder.newBuilder().maximumSize(Integer.MAX_VALUE).build();

    protected PlaceholderResolver placeholderResolver;

    protected ResourceService resourceService;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        return super.allowed(ragConfig, ragData);
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        Assert.hasText(ragConfig.getFile(), "Rag file can not be empty");
        if (log.isDebugEnabled()) {
            log.debug("Rag file start={}", ragConfig.getFile());
        }
        String data = this.placeholderResolver.replace(this.fileCache.get(ragConfig.getFile(), new FileCallable(this.resourceService, ragConfig.getFile())));
        if (log.isDebugEnabled()) {
            log.debug("Rag file key={}, file={}", ragConfig.getFile(), data);
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), data);
        return new RagAtOnce(ragConfig);
    }

    public static class FileCallable implements Callable<String> {

        protected final ResourceService resourceService;

        protected final String file;

        public FileCallable(ResourceService resourceService, String file) {
            this.resourceService = resourceService;
            this.file = file;
        }

        @Override
        public String call() throws Exception {
            // 文件安全性由使用者保证
            return IOUtils.toString(this.resourceService.url(this.file), StandardCharsets.UTF_8);
        }
    }

    @ConditionalOnProperty(name = "file.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Autowired
        protected ResourceService resourceService;

        @Bean(RagFile.RAG_KEY)
        @ConditionalOnMissingBean(name = RagFile.RAG_KEY)
        public RagFile ragFile() throws Exception {
            RagFile ragFile = new RagFile();
            BeanUtils.copyProperties(this, ragFile);
            log.info("RagFile inited, timeout4Condition={}", ragFile.getTimeout4Condition());
            return ragFile;
        }
    }
}
