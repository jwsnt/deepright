package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.listener.EventListenerService;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Builder;
import lombok.Getter;
import org.springframework.util.Assert;

import java.util.Map;

@Builder
@Getter
public class ProviderReaderConfig<T extends ProviderRequest> {

    // 非必须
    protected EventListenerService eventListenerService;

    protected NotifierService notifierService;

    protected LLMCallback llmCallback;

    // 扩展用补充数据
    protected Map<String, Object> extension;

    // Read时最大缓存区
    protected Integer capacity;

    protected Integer discard;

    protected Integer timeout;

    // Read时初始缓存区
    protected Integer buffer;

    protected Integer queue;

    protected T request;

    public ProviderReaderConfig<T> check() throws Exception {
        Assert.notNull(this.notifierService, "The notifier service can not be empty");
        Assert.notNull(this.llmCallback, "The llm callback can not be empty");
        Assert.notNull(this.extension, "The extension can not be empty");
        Assert.notNull(this.capacity, "The capacity can not be empty");
        Assert.notNull(this.discard, "The discard  can not be empty");
        Assert.notNull(this.timeout, "The timeout can not be empty");
        Assert.notNull(this.request, "The request can not be empty");
        Assert.notNull(this.buffer, "The buffer can not be empty");
        Assert.notNull(this.queue, "The queue can not be empty");
        return this;
    }
}
