package ai.deepright.memory;

import ai.deepright.memory.impl.DefMemoryService;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Getter
@Setter
@Slf4j
public class MemoryRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_memory";

    protected MemoryService memoryService;

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.updatePrompt(ragConfig, ragData, this.buildMemory(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected void updatePrompt(RagConfig ragConfig, RagData ragData, Object memory) throws Exception {
        // 当上下文超过10K tokens时，模型对尾部指令的注意力权重下降
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), memory);
    }

    protected String buildMemory(RagConfig ragConfig, RagData ragData) throws Exception {
        String memory = MapUtils.getString(ragData.getQuery().getMetadata(), MemoryRag.RAG_KEY);
        memory = memory != null ? memory : this.memoryService.init(ragData.getQuery());
        ragData.getQuery().putMetadata(MemoryRag.RAG_KEY, memory);
        return StringUtils.defaultIfEmpty(memory, "");
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        @Qualifier(DefMemoryService.NAME)
        protected MemoryService memoryService;

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
