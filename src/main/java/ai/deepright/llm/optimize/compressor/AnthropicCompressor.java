package ai.deepright.llm.optimize.compressor;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.deepright.llm.optimize.FunCallCompressor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.AnyNestedCondition;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Conditional;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class AnthropicCompressor extends StoreCompressor {

    public static final String NAME = FunCallCompressor.FLAG + ProviderRequest.REQUEST_ANTHROPIC;

    @Conditional(CompressCondition.class)
    @Configuration
    public static class CompressInitConfig extends InitConfig {

        @Bean(AnthropicCompressor.NAME)
        @ConditionalOnMissingBean(name = AnthropicCompressor.NAME)
        public AnthropicCompressor anthropicCompressor() throws Exception {
            AnthropicCompressor anthropicCompressor = new AnthropicCompressor();
            BeanUtils.copyProperties(this, anthropicCompressor);
            log.info("AnthropicCompressor inited");
            return anthropicCompressor;
        }
    }

    public static class CompressCondition extends AnyNestedCondition {

        public CompressCondition() {
            super(ConfigurationPhase.REGISTER_BEAN);
        }

        @ConditionalOnProperty(name = "anthropic.enable", havingValue = "true")
        static class OnAnthropicEnable{
        }

        @ConditionalOnProperty(name = "minimax.enable", havingValue = "true")
        static class OnMiniMaxEnable {
        }
    }
}
