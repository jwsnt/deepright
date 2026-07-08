package ai.deepright.llm.optimize.compressor;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.file.FileStore;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.deepright.llm.optimize.FunCallCompressor;
import com.google.common.collect.ImmutableMap;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.AnyNestedCondition;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Conditional;
import org.springframework.context.annotation.Configuration;

import java.nio.charset.StandardCharsets;

@Slf4j
public class OpenAICompressor extends StoreCompressor {

    public static final String NAME = FunCallCompressor.FLAG + ProviderRequest.REQUEST_OPENAI;

    @Override
    public void compress(ProviderRequest providerRequest, LLMFunCallRequest funCallRequest) throws Exception {
        if (this.shouldCompress(funCallRequest)) {
            // String type
            String original = String.class.cast(funCallRequest.getArgs());
            FileStore fileStore = this.defStore.fetchStore(this.store);
            WorkflowException.check(fileStore == null, "The file store can not empty: " + this.store, ProtocolCode.C400);
            // URL压缩在属性中
            String url = this.buildUrl(providerRequest, fileStore.store(original.getBytes(StandardCharsets.UTF_8), ".json", providerRequest.getMessage()));
            funCallRequest.setArgs(JsonUtils.write(ImmutableMap.of("the_original_digest", this.buildRecallQuery(providerRequest, original), "the_complete_content", url)));
            funCallRequest.putMetadata(StoreCompressor.FLAG, true);
            if (log.isWarnEnabled()) {
                log.warn("The request will be compressed, device={}", providerRequest.getMessage().getDevice());
            }
        }
    }

    @Conditional(CompressCondition.class)
    @Configuration
    public static class CompressInitConfig extends InitConfig {

        @Bean(OpenAICompressor.NAME)
        @ConditionalOnMissingBean(name = OpenAICompressor.NAME)
        public OpenAICompressor openAICompressor() throws Exception {
            OpenAICompressor openAICompressor = new OpenAICompressor();
            BeanUtils.copyProperties(this, openAICompressor);
            log.info("OpenAICompressor inited");
            return openAICompressor;
        }
    }

    public static class CompressCondition extends AnyNestedCondition {

        public CompressCondition() {
            super(ConfigurationPhase.REGISTER_BEAN);
        }

        @ConditionalOnProperty(name = "openai.enable", havingValue = "true")
        static class OnOpenAiEnable {
        }

        @ConditionalOnProperty(name = "bigmodel.enable", havingValue = "true")
        static class OnBigModelEnable {
        }

        @ConditionalOnProperty(name = "deepseek.enable", havingValue = "true")
        static class OnDeepseekEnable {
        }

        @ConditionalOnProperty(name = "kimi.enable", havingValue = "true")
        static class OnKimiEnable {
        }

        @ConditionalOnProperty(name = "qwen.enable", havingValue = "true")
        static class OnQwenEnable {
        }

        @ConditionalOnProperty(name = "volcengine.enable", havingValue = "true")
        static class OnVolcengineEnable {
        }
    }
}
