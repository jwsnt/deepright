package ai.open.right.workflow.sync;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.llm.LLMUsage;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.notify.NotifierWriteBack;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import java.util.UUID;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.locks.Condition;
import java.util.concurrent.locks.ReentrantLock;

@Slf4j
public class SyncWriteBack implements NotifierWriteBack {

    public static final Integer MAX_INTERVAL = 10000;

    public static final byte STATUS_SUCCESS = 1;

    public static final byte STATUS_INIT = 0;

    protected final ReentrantLock lock = new ReentrantLock();

    protected final Condition condition = this.lock.newCondition();

    @Getter
    protected final SegmentUsage usage = new SegmentUsage();

    protected final NotifierWriteBack notifierWriteBack;

    protected final SyncCallable syncCallable;

    @Getter
    protected final Integer interval;

    @Getter
    protected final Integer timeout;

    @Getter
    protected final Long created;

    protected final Long start;

    protected byte status = SyncWriteBack.STATUS_INIT;

    // 回调已读长度
    protected Integer startCallable = 0;

    @Getter
    protected String funCallTrack;

    protected Boolean chatTrack = false;

    @Getter
    protected Segment segment;

    @Getter
    @Setter
    // 接管型Fun Call
    protected String takeover;

    public SyncWriteBack(NotifierWriteBack notifierWriteBack, SyncCallable syncCallable, String takeover, Integer interval, Integer timeout, Long created) {
        this.interval = Math.min(interval != null ? interval : timeout, SyncWriteBack.MAX_INTERVAL);
        this.notifierWriteBack = notifierWriteBack;
        this.start = System.currentTimeMillis();
        this.syncCallable = syncCallable;
        this.takeover = takeover;
        this.timeout = timeout;
        this.created = created;
    }

    public SyncWriteBack(NotifierWriteBack notifierWriteBack, String takeover, Integer interval, Integer timeout, Long created) {
        this(notifierWriteBack, null, takeover, interval, timeout, created);
    }

    @Override
    public void beginFunCallTrack(String funCallTrack) {
        this.funCallTrack = funCallTrack;
    }

    @Override
    public void beginFunCallTrack() {
        this.beginFunCallTrack(UUID.randomUUID().toString());
    }


    @Override
    public void beginChatTrack() {
        this.chatTrack = true;
    }

    @Override
    public void closeFunCallTrack() {
        this.funCallTrack = null;
    }

    @Override
    public Boolean containFunCallTrack() {
        return !StringUtils.isEmpty(this.funCallTrack);
    }

    @Override
    public Boolean containChatTrack() {
        return this.chatTrack;
    }

    @Override
    public String getBiz() {
        return this.notifierWriteBack.getBiz();
    }

    @Override
    public String getWorkflow() {
        return this.notifierWriteBack.getWorkflow();
    }

    @Override
    public void ignoreClosed() throws Exception {
        this.notifierWriteBack.ignoreClosed();
    }

    @Override
    public void checkClosed() throws Exception {
        this.notifierWriteBack.checkClosed();
    }

    @Override
    public Boolean isClosed() throws Exception {
        return this.notifierWriteBack.isClosed();
    }

    @Override
    public void close() throws Exception {
        this.notifierWriteBack.close();
    }

    // 回写Callable
    protected void writeCallable(Segment segment) throws Exception {
        if (this.syncCallable != null) {
            // 复制并标记Start（独立计数）
            Segment copy = segment.copyWithStart(this.startCallable);
            try {
                this.syncCallable.call(copy);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
            copy.mark();
            this.startCallable = copy.getStart();
        }
    }

    @Override
    public void writeSource(Segment segment) throws Exception {
        this.notifierWriteBack.writeSource(segment);
    }

    @Override
    public void writeBack(Segment segment) throws Exception {
        this.lock.lockInterruptibly();
        try {
            LLMUsage usage = segment.getUsage();
            if (usage != null) {
                // 异常类，特殊Segment没有Usage
                this.usage.addUsage(usage);
            }
            // 如果是TakeOver流量，则复制推送一份（最优）
            if (!StringUtils.isEmpty(this.takeover)) {
                this.notifierWriteBack.writeBack(segment.copy());
            }
            // 回调线程
            this.writeCallable(segment);
            // 当前线程
            if (segment.isFinished()) {
                this.status = SyncWriteBack.STATUS_SUCCESS;
                this.segment = segment;
                this.condition.signalAll();
            }
        } finally {
            this.lock.unlock();
        }
    }

    public String get() throws Exception {
        this.lock.lockInterruptibly();
        try {
            while (SyncWriteBack.STATUS_INIT == this.status) {
                this.checkClosed();
                // 计算Await超时
                if (System.currentTimeMillis() > (this.start + this.timeout)) {
                    throw new WorkflowException("The sync write back (" + SplitUtils.join(this.getWorkflow(), this.getBiz()) + ") read timeout: " + this.timeout, ProtocolCode.C502);
                }
                if (this.condition.await(this.interval, TimeUnit.MILLISECONDS) && log.isDebugEnabled()) {
                    log.debug("The sync condition is signaled");
                }
            }
        } finally {
            this.lock.unlock();
        }
        // Change Protocol Code
        if (!ProtocolCode.range2xx(this.segment.getCode())) {
            WorkflowException exception = new WorkflowException(this.segment.getContent(), this.segment.getCode());
            // 如果Code小于等于0，静默异常
            // 900静默
            exception = (ProtocolCode.C0 >= this.segment.getCode() || ProtocolCode.rangeCode(this.segment.getCode(), ProtocolCode.C900)) ? exception.needSilent() : exception;
            throw exception;
        }
        if (log.isDebugEnabled()) {
            log.debug("The sync segment {}", this.segment);
        }
        // 不能Trim，防止破坏Markdown
        return StringUtils.defaultIfEmpty(this.segment.getContent(), "");
    }

    public <T> T get(Class<T> clazz) throws Exception {
        return (T) JsonUtils.read(this.get(), clazz);
    }
}
