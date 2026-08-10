package ai.open.right.workflow.notify;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.Segment;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

import java.util.UUID;

public class NotifierWriteBase implements NotifierWriteBack {

    protected Boolean ignoreClosed = false;

    protected String funCallTrack;

    protected Boolean chatTrack = false;

    @Getter
    @Setter
    // 接管型Fun Call的Notifier
    protected String takeover;

    @Setter
    protected String workflow;

    protected Boolean closed = false;

    @Setter
    protected String biz;

    @Getter
    @Setter
    protected Long created;

    @Override
    public void beginFunCallTrack(String funCallTrack) {
        this.funCallTrack = funCallTrack;
    }

    @Override
    public void beginFunCallTrack() {
        this.beginFunCallTrack(UUID.randomUUID().toString());
    }

    @Override
    public void closeFunCallTrack() {
        this.funCallTrack = null;
    }

    @Override
    public String getFunCallTrack() {
        return this.funCallTrack;
    }

    @Override
    public void beginChatTrack() {
        this.chatTrack = true;
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
        return this.biz;
    }

    @Override
    public String getWorkflow() {
        return this.workflow;
    }

    @Override
    public void writeSource(Segment segment) throws Exception {
    }

    @Override
    public void writeBack(Segment segment) throws Exception {
    }

    @Override
    public void ignoreClosed() throws Exception {
        this.ignoreClosed = true;
    }

    @Override
    public void checkClosed() throws Exception {
        if (Boolean.TRUE.equals(this.ignoreClosed)) {
            return;
        }
        if (this.closed) {
            // 静默的异常，标记被动关闭
            throw new WorkflowException("The task (notifier) was closed", ProtocolCode.CN1).needSilent();
        }
    }

    @Override
    public Boolean isClosed() throws Exception {
        return this.closed;
    }

    @Override
    public void close() throws Exception {
        this.closed = true;
    }
}
