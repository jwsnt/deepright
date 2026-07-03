package ai.deepright.llm.optimize.compressor;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRouter;
import ai.deepright.llm.optimize.FunCallCompressor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.AnyNestedCondition;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Conditional;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

@Slf4j
public class GoogleCompressor extends StoreCompressor {

    public static final String NAME = FunCallCompressor.FLAG + ProviderRequest.REQUEST_GOOGLE;

    @Override
    public void compress(ProviderRequest providerRequest, LLMFunCallRequest funCallRequest) throws Exception {
        if (this.shouldCompress(funCallRequest)) {
            // 压缩，替换为固定思考
            funCallRequest.getRefer().put("thoughtSignature", GoogleRouter.GoogleMessage.GooglePart.SIGNATURE);
            super.compress(providerRequest, funCallRequest);
        }
    }

    @Conditional(GoogleCondition.class)
    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    public static class CompressInitConfig extends InitConfig {

        @Bean(GoogleCompressor.NAME)
        @ConditionalOnMissingBean(name = GoogleCompressor.NAME)
        public GoogleCompressor googleCompressor() throws Exception {
            GoogleCompressor googleCompressor = new GoogleCompressor();
            BeanUtils.copyProperties(this, googleCompressor);
            log.info("GoogleCompressor inited");
            return googleCompressor;
        }
    }

    public static class GoogleCondition extends AnyNestedCondition {

        public GoogleCondition() {
            super(ConfigurationPhase.REGISTER_BEAN);
        }

        @ConditionalOnProperty(name = "gemini.enable", havingValue = "true")
        static class OnGoogleEnable {
        }

        @ConditionalOnProperty(name = "vertex.enable", havingValue = "true")
        static class OnVertexEnable {
        }
    }
}
