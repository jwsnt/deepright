package ai.deepright.llm;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderCallback;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;

import java.util.concurrent.BlockingQueue;

@Slf4j
public class RetryCallback extends ProviderReaderCallback {

    protected final RetryConfig retryConfig;

    protected volatile Boolean failed = false;

    public RetryCallback(ProviderReaderConfig<ProviderRequest> providerReaderConfig, BlockingQueue<Object> messageQueue, ProviderRequest request, WorkflowTask workTask) throws Exception {
        super(providerReaderConfig, messageQueue, request, workTask);
        RetryConfig retryConfig = RetryConfig.class.cast(MapUtils.getObject(providerReaderConfig.getExtension(), RetryUtils.RETRY));
        WorkflowException.check(retryConfig == null, "The retry config can not be empty", ProtocolCode.C400);
        this.retryConfig = retryConfig.check();
    }

    @Override
    protected void notifyException(WorkflowException exception) throws Exception {
        // 未失败过 且重试成功
        if (!this.failed && RetryUtils.isRetry(this.notifierService, exception, this.request, this.retryConfig)) {
            this.failed = true;
        } else {
            super.notifyException(exception);
        }
    }

    @Override
    protected void callback(String message) throws Exception {
        if (!this.failed) {
            super.callback(message);
        } else if (log.isDebugEnabled()) {
            log.debug("The read `callback` will be ignored when failed={}", this.failed);
        }
    }
}
