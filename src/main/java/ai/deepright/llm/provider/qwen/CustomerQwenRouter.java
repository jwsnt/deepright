package ai.deepright.llm.provider.qwen;

import ai.deepright.llm.RetryConfig;
import ai.deepright.llm.RetryUtils;
import ai.deepright.llm.optimize.rag.RequestModelRag;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.llm.provider.openai.CustomerOpenAiReader;
import ai.open.right.workflow.flow.WorkflowQueue;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.qwen.QwenRouter;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.entity.StringEntity;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

import java.util.concurrent.ScheduledExecutorService;

@Slf4j
@Getter
@Setter
public class CustomerQwenRouter extends QwenRouter {

    protected ScheduledExecutorService scheduled;

    protected WorkflowQueue workflowQueue;

    protected Integer mediaCapacity;

    protected Integer sleep;

    protected Integer retry;

    @Override
    public OpenAiReader reader(OpenAiRequest request, LLMConfig llmConfig, LLMCallback llmCallback) throws Exception {
        return new CustomerOpenAiReader(ProviderReaderConfig.<OpenAiRequest>builder()
                .capacity(RequestModelSelect.multiOutput(request.getMessage()) ? this.mediaCapacity : this.capacity)
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

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Getter
    @Setter
    public static class CustomerInitConfig extends InitConfig {

        @Autowired
        protected ScheduledExecutorService scheduled;

        @Autowired
        protected WorkflowQueue workflowQueue;

        @Value("${request.capacity.media:104857600}")
        protected Integer mediaCapacity;

        @Value("${request.provider.sleep:45000}")
        protected Integer sleep;

        @Value("${request.provider.retry:3}")
        protected Integer retry;

        @Override
        @Bean(QwenRouter.NAME)
        public QwenRouter qwenRouter() {
            CustomerQwenRouter qwenRouter = new CustomerQwenRouter();
            BeanUtils.copyProperties(this, qwenRouter);
            log.info("CustomerQwenModelRouter inited");
            return qwenRouter;
        }
    }
}
