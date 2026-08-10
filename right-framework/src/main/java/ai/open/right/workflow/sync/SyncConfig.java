package ai.open.right.workflow.sync;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.media.MediaContext;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Getter
@Setter
@Builder
public class SyncConfig {

    public static final Integer INTERVAL = 5000;

    protected List<MediaContext> mediaContext;

    // 不会修改WorkflowTask的Metadata
    protected Map<String, Object> metadata;

    protected SyncCallable syncCallable;

    @Setter
    protected WorkflowTask workTask;

    protected String workflow;

    @Setter
    @Builder.Default
    protected Integer interval = SyncConfig.INTERVAL;

    @Setter
    protected Integer timeout;

    // 接管型Fun Call的Notifier
    protected String takeover;

    protected String notifier;

    // 选择模型供应商
    protected String provider;

    protected String reQuery;

    @Builder.Default
    // 不继承metadata
    protected Boolean pure = false;

    protected String chat;

    protected String biz;

    public Map<String, Object> getMetadata() {
        if (!StringUtils.isEmpty(this.provider)) {
            // 追加Provider
            this.metadata = !CollectionUtils.isEmpty(this.metadata) ? this.metadata : new HashMap<String, Object>();
            this.metadata.put(ProviderRequestService.KEY_PROVIDER, this.provider);
        }
        return this.metadata;
    }
}
