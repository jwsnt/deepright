package ai.deepright.llm;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.notifier.MultiSourceFlag;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import com.google.common.collect.ImmutableMap;
import lombok.Builder;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;

import java.util.List;
import java.util.Objects;
import java.util.concurrent.TimeUnit;

@Slf4j
public class RetryUtils {

    public static final String LANG_KEY_RETRY_MESSAGE = "retry.message";

    public static final String RETRY = "retry";

    // 重新放入队列
    public static Boolean isRetry(NotifierService notifierService, Exception exception, ProviderRequest request, RetryConfig config) throws Exception {
        try {
            // 重试计数（单Loop返回内的重试）
            Integer tried = request.getMessage().getMetadata(RetryUtils.RETRY, Integer.class);
            tried = tried == null ? 0 : tried;
            if (log.isInfoEnabled()) {
                log.info("The request has been retried {} times", tried);
            }
            // 静默型异常和401不重试
            Integer code = WorkflowException.code(exception);
            if (code >= ProtocolCode.C429 && tried < config.getRetry() && !WorkflowException.silent(exception)) {
                int delay = ProtocolCode.C429.equals(code) ? config.getSleep() : config.getSleep() / 2;
                RetryUtils.notify(notifierService, request.getMessage(), XmlResourceLang.get(RetryUtils.LANG_KEY_RETRY_MESSAGE).replace("#code", String.valueOf(code)).replace("#delay", String.valueOf(delay / 1000)), code, delay);
                config.getScheduled().schedule(RetryRunnable.builder()
                        .notifierService(notifierService)
                        .exception(exception)
                        .request(request)
                        .config(config)
                        .tried(tried)
                        .build(), delay, TimeUnit.MILLISECONDS);
                return true;
            } else {
                RetryUtils.clean(request.getMessage());
                return false;
            }
        } catch (Exception ex) {
            RetryUtils.clean(request.getMessage());
            WorkflowException.dolog(ex);
            return false;
        }
    }

    public static void storeQuery(ProviderRequest request, LLMConfig llmConfig, HistoryStore historyStore, List<HistoryPair> historyPairs) throws Exception {
        // 如果需要单独存储Request，逻辑需要与ProviderStream一致
        if (ProviderRequestService.shouldStoreHistoryQuery(request, llmConfig) && MapUtils.getObject(request.getMessage().getMetadata(), RETRY) == null) {
            // 不是Retry请求才记录历史记录
            historyStore.store(request.getMessage(), request.getRepositories(), historyPairs, request.getExpired(), request.getHistories());
        }
    }

    // 推送回端
    public static void notify(NotifierService notifierService, WorkflowTask workTask, String content, Integer code, Integer delay) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                    // 推送空格保持连接
                    .metadata(ImmutableMap.of(MultiSourceFlag.RETRY, code, MultiSourceFlag.DELAY, delay))
                    .content(new StringBuffer(CliPrinter.format(content, CliPrinter.SIZE_N)))
                    .workflow(workTask.getWorkflow())
                    .notifier(Notifier.SOURCE)
                    .build();
            notifierService.notify(Segment.build(workTask, segmentConfig), workTask, workTask);
        }
    }

    // 重配置
    public static void config(ProviderRequest request, Integer tried) throws Exception {
        request.getMessage().putMetadata(RetryUtils.RETRY, tried + 1);
        // 删除所有非Client会话（重置）
        List<History> histories = request.getMessage().getHistories();
        if (!CollectionUtils.isEmpty(histories)) {
            histories.removeIf(history -> !Objects.equals(history.getReference(), History.REFERENCE_CLIENT));
        }
        request.getMessage().setHistories(histories);
    }

    public static void clean(WorkflowTask workTask) throws Exception {
        workTask.delMetadata(RetryUtils.RETRY);
    }

    @Builder
    public static class RetryRunnable implements Runnable {

        protected NotifierService notifierService;

        protected ProviderRequest request;

        protected Exception exception;

        protected RetryConfig config;

        protected Integer tried;

        @Override
        public void run() {
            try {
                RetryUtils.config(this.request, this.tried);
                this.config.getWorkflowQueue().put(this.request.getMessage());
                if (log.isInfoEnabled()) {
                    log.info("The request will be reput to the queue, tried={}", this.tried);
                }
            } catch (Exception ex) {
                WorkflowException.dolog(ex);
            }
        }
    }
}
