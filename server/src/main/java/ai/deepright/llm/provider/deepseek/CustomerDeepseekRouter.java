package ai.deepright.llm.provider.deepseek;

import ai.deepright.llm.RetryConfig;
import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.optimize.rag.RequestModelRag;
import ai.deepright.llm.provider.openai.CustomerOpenAiReader;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.deepseek.DeepSeekRouter;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.entity.StringEntity;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.concurrent.ScheduledExecutorService;

@Slf4j
@Getter
@Setter
public class CustomerDeepseekRouter extends DeepSeekRouter {

    protected ScheduledExecutorService scheduled;

    protected WorkflowQueue workflowQueue;

    protected Integer sleep;

    protected Integer retry;

    @Override
    public OpenAiReader reader(OpenAiRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new CustomerOpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .buffer(llmConfig.hasNetworkBuffer() ? llmConfig.getNetworkBuffer() : this.buffer)
                .extension(ImmutableMap.of(RetryUtils.RETRY, RetryConfig.builder()
                        .workflowQueue(this.workflowQueue)
                        .scheduled(this.scheduled)
                        .retry(this.retry)
                        .sleep(this.sleep)
                        .build()))
                .eventListenerService(this.eventListenerService)
                .notifierService(this.notifierService)
                .timeout(this.queueTimeout)
                .llmCallback(llmCallback)
                .capacity(this.capacity)
                .discard(this.discard)
                .queue(this.queue)
                .request(request).build().check());
    }

    @Override
    protected StringEntity buildEntity(OpenAiRequest request, LLMConfig llmConfig) throws Exception {
        StringEntity entity = super.buildEntity(request, llmConfig);
        request.getMessage().putMetadata(RequestModelRag.LANG_KEY_REQUEST_CAPACITY, entity.getContentLength());
        request.getMessage().putMetadata(RequestModelRag.LANG_KEY_REQUEST_MODEL, request.getModel());
        return entity;
    }

    @ConditionalOnProperty(name = "deepseek.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Getter
    @Setter
    public static class CustomerInitConfig extends InitConfig {

        @Autowired
        protected ScheduledExecutorService scheduled;

        @Autowired
        protected WorkflowQueue workflowQueue;

        @Value("${request.provider.sleep:45000}")
        protected Integer sleep;

        @Value("${request.provider.retry:3}")
        protected Integer retry;

        @Override
        @Bean(DeepSeekRouter.NAME)
        @ConditionalOnMissingBean(name = DeepSeekRouter.NAME)
        public DeepSeekRouter deepSeekRouter() {
            CustomerDeepseekRouter deepSeekRouter = new CustomerDeepseekRouter();
            BeanUtils.copyProperties(this, deepSeekRouter);
            log.info("CustomerDeepseekModelRouter inited");
            return deepSeekRouter;
        }
    }
}
