package ai.open.right.workflow.notify;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.Segment;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class NothingWriteBack implements NotifierWriteBack {

    protected Boolean ignoreClosed = false;

    protected Boolean closed = false;

    protected String takeover;

    protected String workflow;

    protected String biz;

    @Override
    public void beginFunCallTrack(String funCallTrack) {
    }

    @Override
    public void beginFunCallTrack() {
    }

    @Override
    public void beginChatTrack() {
    }

    @Override
    public void closeFunCallTrack() {
    }

    @Override
    public String getFunCallTrack() {
        return null;
    }

    @Override
    public Boolean containFunCallTrack() {
        return false;
    }

    @Override
    public Boolean containChatTrack() {
        return false;
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
        if (!this.getIgnoreClosed() && this.getClosed()) {
            // 静默异常，标记被动关闭
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

