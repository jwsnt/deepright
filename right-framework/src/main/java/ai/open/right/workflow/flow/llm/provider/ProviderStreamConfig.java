package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Builder;
import lombok.Getter;
import org.springframework.util.Assert;

import java.util.Map;

@Builder
@Getter
public class ProviderStreamConfig<T extends ProviderRequest> {

    // 非必选
    protected ProviderStorePolicy providerStorePolicy;

    // 非必选
    protected TrackFunCallService trackFunCallService;

    protected MediaInlineService mediaInlineService;

    protected NotifierService notifierService;

    protected TokenStatistic tokenStatistic;

    protected ProviderReason providerReason;

    // 扩展用补充数据
    protected Map<String, Object> extension;

    // 非必选
    protected SignalStream signalStream;

    protected HistoryStore historyStore;

    protected NamesService namesService;

    protected T request;

    public ProviderStreamConfig<T> check() throws Exception {
        Assert.notNull(this.notifierService, "The notifier service can not be empty");
        Assert.notNull(this.historyStore, "The history store can not be empty");
        Assert.notNull(this.namesService, "The names service can not be empty");
        Assert.notNull(this.request, "The request can not be empty");
        return this;
    }
}
