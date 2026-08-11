package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.concurrent.FutureCallback;
import org.springframework.util.Assert;

import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

@Slf4j
@Getter
public class ProviderReaderCallback implements Runnable, FutureCallback<Void> {

    protected static final AtomicInteger RUNNER_COUNTER = new AtomicInteger(0);

    // 用于终止消息
    protected static final Object CLOSED = new Object();

    protected final BlockingQueue<Object> messageQueue;

    protected final NotifierService notifierService;

    protected final LLMCallback llmCallback;

    protected final ProviderRequest request;

    protected final WorkflowTask workTask;

    protected final Integer timeout;

    protected final Integer discard;

    protected volatile Boolean released = false;

    protected volatile Boolean notified = false;

    protected volatile Boolean failed = false;

    public ProviderReaderCallback(ProviderReaderConfig<ProviderRequest> providerReaderConfig, BlockingQueue<Object> messageQueue, ProviderRequest request, WorkflowTask workTask) {
        Assert.notNull(providerReaderConfig.getDiscard(), "The discard can not be empty");
        Assert.notNull(providerReaderConfig.getTimeout(), "The timeout can not be empty");
        this.notifierService = providerReaderConfig.getNotifierService();
        this.llmCallback = providerReaderConfig.getLlmCallback();
        this.discard = providerReaderConfig.getDiscard();
        this.timeout = providerReaderConfig.getTimeout();
        this.messageQueue = messageQueue;
        this.workTask = workTask;
        this.request = request;
    }

    public void released() {
        this.released = true;
    }

    @Override
    public void run() {
        try {
            ProviderReaderCallback.RUNNER_COUNTER.incrementAndGet();
            int idle = 0;
            while (!this.released && !this.isClosed()) {
                Object message = this.messageQueue.poll(this.timeout, TimeUnit.MILLISECONDS);
                if (message != null) {
                    try {
                        if (log.isDebugEnabled()) {
                            log.debug("The response message is processing={}", message);
                        }
                        if (!ProviderReaderCallback.CLOSED.equals(message)) {
                            this.callback(String.class.cast(message));
                        } else {
                            if (log.isDebugEnabled()) {
                                log.debug("The response queue was closed");
                            }
                            break;
                        }
                    } finally {
                        idle = 0;
                    }
                } else if (this.discard > 0 && (idle += this.timeout) >= this.discard) {
                    if (log.isWarnEnabled()) {
                        log.warn("The response will be discarded, idle={} >= discard={}", idle, this.discard);
                    }
                    this.released();
                    throw new WorkflowException("The response will be discarded, idle=" + idle);
                }
            }
            this.success();
        } catch (Exception e) {
            // 任何异常立即终止
            this.failed(e);
        } finally {
            ProviderReaderCallback.RUNNER_COUNTER.decrementAndGet();
        }
    }

    // 将异常信息回写
    protected void notifyException(WorkflowException exception) throws Exception {
        if (!this.notified) {
            // 仅通知一次
            this.notifierService.notify(Segment.failed(this.workTask, exception.getMessage(), this.workTask.getNotifier(), exception.getCode()), this.workTask);
            this.notified = true;
        } else if (log.isErrorEnabled()) {
            WorkflowException.dolog(exception);
        }
    }

    protected void callback(String message) throws Exception {
        this.llmCallback.callback(message);
    }

    protected Boolean isClosed() throws Exception {
        return this.request.getMessage().isClosed();
    }

    protected void success() throws Exception {
        if (!this.failed) {
            this.request.autoDump();
        }
    }

    @Override
    public void completed(Void result) {
        if (log.isDebugEnabled()) {
            log.debug("The request completed");
        }
    }

    @Override
    public void failed(Exception ex) {
        try {
            if (log.isWarnEnabled()) {
                log.warn("The request={} failed={}", this.request.getUrl(), ex.getMessage());
            }
            this.failed = true;
            Integer code = WorkflowException.code(ex, ProtocolCode.C800);
            String message = ProtocolCode.C400.equals(code) ? this.request.getProviderData().getResponse() : ex.getMessage();
            this.notifyException(WorkflowException.create(message, code).needSilent(ex));
        } catch (Exception e) {
            WorkflowException.dolog(e);
        } finally {
            // 自动DUMP
            this.request.autoDump(WorkflowException.create(ex));
            this.released();
        }
    }

    public void cancelled() {
        if (log.isInfoEnabled()) {
            log.info("The request cancelled");
        }
        // 释放队列并通知
        this.failed = true;
        this.released();
    }
}