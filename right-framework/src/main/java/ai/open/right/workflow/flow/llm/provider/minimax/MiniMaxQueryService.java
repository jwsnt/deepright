package ai.open.right.workflow.flow.llm.provider.minimax;

import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicQueryService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Setter
@Getter
@Slf4j
public class MiniMaxQueryService extends AnthropicQueryService {

    protected MiniMaxRequestService miniMaxRequestService;

    protected MiniMaxRouter miniMaxRouter;

    @Override
    protected MiniMaxRequestService request() {
        return this.miniMaxRequestService;
    }

    @Override
    protected MiniMaxRouter router() {
        return this.miniMaxRouter;
    }

    @ConditionalOnProperty(name = "minimax.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderQueryInitConfig {

        @Autowired
        protected MiniMaxRequestService miniMaxRequestService;

        @Autowired
        protected MiniMaxRouter miniMaxRouter;

        @Bean(name = LLMQueryService.LLM_MINIMAX)
        @ConditionalOnMissingBean(name = LLMQueryService.LLM_MINIMAX)
        public MiniMaxQueryService miniMaxQueryService() throws Exception {
            MiniMaxQueryService miniMaxQueryService = new MiniMaxQueryService();
            BeanUtils.copyProperties(this, miniMaxQueryService);
            log.info("MiniMaxQueryService inited");
            return miniMaxQueryService;
        }
    }
}
