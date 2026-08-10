package ai.open.right.integration;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.protocol.Protocol;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowObject;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.sync.SyncCallable;
import ai.open.right.workflow.sync.SyncConfig;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Builder
@Setter
@Getter
// @See WorkflowTask，属性等价
public class RightConfig implements WorkflowObject {

    protected NotifierWriteBack notifierWriteBack;

    protected List<MediaContext> mediaContext;

    @Builder.Default
    private Map<String, Object> metadata = new HashMap<String, Object>();

    // 用于异步Callback Write
    protected SyncCallable syncCallable;

    protected List<History> histories;

    protected UserContext userContext;

    protected String conversation;

    // 是否开启Fun Call Track的ID
    protected String funCallTrack;

    // 是否开启Chat Track
    @Builder.Default
    protected Boolean chatTrack = false;

    protected String markQuery;

    protected Integer deepness;

    @Builder.Default
    protected Integer interval = SyncConfig.INTERVAL;

    protected Integer timeout;

    protected String notifier;

    @Builder.Default
    protected String protocol = Protocol.CHAT;

    protected String upstream;

    // 接管型Fun Call的Notifier
    protected String takeover;

    protected String workflow;

    protected String provider;

    protected String trace;

    protected String query;

    protected String chat;

    protected String biz;

    public RightConfig init() {
        this.upstream = this.workflow;
        return this;
    }

    public Integer getTimeout(Integer timeout) {
        return this.timeout != null ? this.timeout : timeout;
    }

    public void setHistories(List<History> histories) {
        this.histories = histories != null ? new ArrayList<History>(histories) : null;
    }

    @Override
    public void setObjectQuery(Object object) throws Exception {
        this.setQuery(JsonUtils.write(object));
    }

    @Override
    public <T> T getObjectQuery(Class<T> clazz) throws Exception {
        return JsonUtils.read(this.getQuery(), clazz);
    }

    public Map<String, Object> getMetadata() {
        if (!StringUtils.isEmpty(this.provider)) {
            // 追加Provider
            this.metadata = !CollectionUtils.isEmpty(this.metadata) ? this.metadata : new HashMap<String, Object>();
            this.metadata.put(ProviderRequestService.KEY_PROVIDER, this.provider);
        }
        return this.metadata;
    }

    public List<History> getHistories() {
        // 被动创建
        this.histories = this.histories != null ? this.histories : new ArrayList<History>();
        return this.histories;
    }

    public RightConfig incrDeepness() {
        if (this.deepness != null) {
            this.deepness = this.deepness + RedirectContext.DEEPNESS;
        } else {
            this.deepness = RedirectContext.DEEPNESS;
        }
        return this;
    }
}
