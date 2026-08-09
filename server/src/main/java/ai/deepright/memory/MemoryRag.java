package ai.deepright.memory;

import ai.deepright.feature.FeatureFlag;
import ai.deepright.memory.impl.DefMemoryService;
import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;

@Getter
@Setter
@Slf4j
public class MemoryRag extends RagCondition implements RagService {

    public static final String KEY_READONLY = "#knowledge_readonly";

    public static final String RAG_KEY = "rag_memory";

    protected ResourceService resourceService;

    protected MemoryService memoryService;

    protected String template4readonly;

    @PostConstruct
    public void init() throws Exception {
        this.template4readonly = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4readonly).openStream()), StandardCharsets.UTF_8);
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4readonly), "The template readonly must not be empty");
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.updateLongMemory(ragConfig, ragData, this.buildMemory(ragConfig, ragData));
        this.updateReadonly(ragConfig, ragData);
        return new RagAtOnce(ragConfig);
    }

    @Override
    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        // 跳过测试请求
        return super.allowed(ragConfig, ragData) && !FeatureFlag.isTest(ragData.getQuery());
    }

    protected void updateLongMemory(RagConfig ragConfig, RagData ragData, Object memory) throws Exception {
        // 当上下文超过10K tokens时，模型对尾部指令的注意力权重下降
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), memory);
    }

    protected void updateReadonly(RagConfig ragConfig, RagData ragData) throws Exception {
        RagService.updatePrompt(ragConfig, ragData, MemoryRag.KEY_READONLY, !FeatureFlag.isKnowledgeCommit(ragData.getQuery()) ? this.template4readonly : "");
    }

    protected String buildMemory(RagConfig ragConfig, RagData ragData) throws Exception {
        String memory = MapUtils.getString(ragData.getQuery().getMetadata(), MemoryRag.RAG_KEY);
        memory = memory != null ? memory : this.memoryService.init(ragData.getQuery());
        ragData.getQuery().putMetadata(MemoryRag.RAG_KEY, memory);
        return StringUtils.defaultIfEmpty(memory, "");
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        @Qualifier(DefMemoryService.NAME)
        protected MemoryService memoryService;

        @Value("${memory.knowledge.update:classpath:config/memory/knowledge_readonly.md}")
        protected String template4readonly;

        @Bean(MemoryRag.RAG_KEY)
        @ConditionalOnMissingBean(name = MemoryRag.RAG_KEY)
        public MemoryRag memoryRag() throws Exception {
            MemoryRag memoryRag = new MemoryRag();
            BeanUtils.copyProperties(this, memoryRag);
            log.info("MemoryRag inited, timeout4Condition={}", memoryRag.getTimeout4Condition());
            return memoryRag;
        }
    }
}
