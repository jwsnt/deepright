package ai.open.right.workflow.notify;

import ai.open.right.workflow.flow.llm.Segment;

public interface NotifierWriteBack extends NotifierTrack {

    public String getWorkflow();

    // TakeOver的Notifier模式
    public String getTakeover();

    public String getBiz();

    public void setTakeover(String takeover);

    // 写入顶层通道（通常为调用端）
    public void writeSource(Segment segment) throws Exception;

    // 写入上层通道（发起Chain的）
    public void writeBack(Segment segment) throws Exception;

    // 执行时忽略通道是否关闭
    public void ignoreClosed() throws Exception;

    // 是否已关闭
    public void checkClosed() throws Exception;

    public Boolean isClosed() throws Exception;

    public void close() throws Exception;
}
