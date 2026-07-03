package ai.deepright.media;

import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderImageConfig;
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
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
public class MediaRag extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_media";

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.updateConfig(ragData);
        return new RagAtOnce(ragConfig);
    }

    protected void updateConfig(RagData ragData) throws Exception {
        if (ProviderImageConfig.class.isAssignableFrom(ragData.getRequest().getClass())) {
            // 使用原始模型
            ProviderImageConfig.class.cast(ragData.getRequest()).setImageConfig(MapUtils.getMap(MapUtils.getMap(ragData.getQuery().getMetadata(), FeatureField.KEY_MEDIA), FeatureUtils.buildSourceProvider(ragData.getQuery())));
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends RagCondition.ConditionInitConfig {

        @Bean(MediaRag.RAG_KEY)
        @ConditionalOnMissingBean(name = MediaRag.RAG_KEY)
        public MediaRag mediaRag() throws Exception {
            MediaRag mediaRag = new MediaRag();
            BeanUtils.copyProperties(this, mediaRag);
            log.info("MediaRag inited, timeout4Condition={}", mediaRag.getTimeout4Condition());
            return mediaRag;
        }
    }
}
