package ai.open.right.workflow.flow.llm.rag.meta;

import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlElementWrapper;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlProperty;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlRootElement;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlText;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Slf4j
// 使用Metadata增强内容
public class RagMeta extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_meta";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag meta start");
        }
        RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildMetadata(ragConfig, ragData));
        return new RagAtOnce(ragConfig);
    }

    protected Map<String, Object> metadata(RagConfig ragConfig, RagData ragData, Map<String, Object> metadata) throws Exception {
        if (!MapUtils.isEmpty(metadata) && ragConfig.hasRagMeta()) {
            // 过滤掉不允许使用的Metadata
            return metadata.entrySet().stream()
                    .filter(entry -> ragConfig.getRagMetaConfig().allowed(entry.getKey()))
                    .collect(Collectors.toMap(Map.Entry::getKey, Map.Entry::getValue));
        } else {
            return metadata;
        }
    }

    protected Object buildMetadata(RagConfig ragConfig, RagData ragData) throws Exception {
        Map<String, Object> metadata = this.metadata(ragConfig, ragData, ragData.getQuery().getMetadata());
        if (ragConfig.isMode(RagConfig.MODE_JSON) || CollectionUtils.isEmpty(metadata)) {
            return metadata;
        }
        LLMMetadataPrompts metaPrompts = new LLMMetadataPrompts();
        for (String key : metadata.keySet()) {
            LLMItemPrompts input = LLMItemPrompts.builder()
                    .content(metadata.get(key))
                    .mcode(key)
                    .build();
            metaPrompts.add(input);
        }
        if (log.isDebugEnabled()) {
            log.debug("The rag metadata={}", metaPrompts);
        }
        return metaPrompts;
    }

    @Getter
    @JacksonXmlRootElement(localName = "Metadata")
    public static class LLMMetadataPrompts {

        @JacksonXmlElementWrapper(useWrapping = false)
        @JacksonXmlProperty(localName = "Item")
        protected List<LLMItemPrompts> item;

        public LLMMetadataPrompts add(LLMItemPrompts input) {
            if (this.item == null) {
                this.item = new ArrayList<LLMItemPrompts>();
            }
            this.item.add(input);
            return this;
        }
    }

    @Getter
    @Builder
    @JacksonXmlRootElement(localName = "Item", namespace = "")
    public static class LLMItemPrompts {

        @JacksonXmlText
        protected Object content;

        @JacksonXmlText
        protected String mcode;
    }

    @ConditionalOnProperty(name = "meta.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Bean(RagMeta.RAG_KEY)
        @ConditionalOnMissingBean(name = RagMeta.RAG_KEY)
        public RagMeta ragMeta() throws Exception {
            RagMeta ragMeta = new RagMeta();
            BeanUtils.copyProperties(this, ragMeta);
            log.info("RagMeta inited, timeout4Condition={}", ragMeta.getTimeout4Condition());
            return ragMeta;
        }
    }
}

