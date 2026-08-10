package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;

@Setter
@Getter
@Slf4j
abstract public class ProviderQueryService<T extends ProviderRequest> implements LLMQueryService {

    protected ProviderStorePolicy providerStorePolicy;

    protected TrackFunCallService trackFunCallService;

    protected MediaInlineService mediaInlineService;

    protected NotifierService notifierService;

    protected TokenStatistic tokenStatistic;

    protected ProviderReason providerReason;

    protected NamesService namesService;

    protected HistoryStore historyStore;

    abstract protected ProviderStream<T> stream(SignalStream signalStream, T r) throws Exception;

    abstract protected ProviderRequestService<T> request() throws Exception;

    abstract protected ProviderRouter<T> router() throws Exception;

    @Override
    public void query(LLMQuery llmQuery, LLMConfig llmConfig, SignalStream signalStream) throws Exception {
        LLMQuery.LLMQueryChecker.check(llmQuery);
        // 配置Request
        T request = this.request().config(llmConfig, llmQuery);
        // 配置Router
        this.router().route(request, llmConfig, this.stream(signalStream, request));
    }

    @Override
    public void query(LLMQuery llmQuery, LLMConfig llmConfig) throws Exception {
        this.query(llmQuery, llmConfig, SignalStream.EMPTY);
    }

    @Setter
    @Getter
    public static class ProviderQueryInitConfig {

        @Autowired(required = false)
        protected ProviderStorePolicy providerStorePolicy;

        @Autowired(required = false)
        protected TrackFunCallService trackFunCallService;

        @Autowired(required = false)
        protected MediaInlineService mediaInlineService;

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected ProviderReason providerReason;

        @Autowired
        protected TokenStatistic tokenStatistic;

        @Autowired(required = false)
        protected HistoryStore historyStore;

        @Autowired
        protected NamesService namesService;

    }
}
