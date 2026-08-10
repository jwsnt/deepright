package ai.open.right.workflow.flow.llm.rag.remote;

import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Getter;

import java.util.Map;

@Getter
// 不需要清理且为Post则自动转为Json（用于Post的标准请求）
public class RagRequest {

    protected final Map<String, Object> metadata;

    protected final UserContext userContext;

    protected final String conversation;

    protected final String workflow;

    protected final Long timestamp;

    protected final String trace;

    protected final Object query;

    protected final String chat;

    protected final String biz;

    public RagRequest(WorkflowTask workTask) {
        this.conversation = workTask.getConversation();
        this.userContext = workTask.getUserContext();
        this.timestamp = workTask.getCreated();
        this.metadata = workTask.getMetadata();
        this.workflow = workTask.getWorkflow();
        this.trace = workTask.getTrace();
        this.query = workTask.getQuery();
        this.chat = workTask.getChat();
        this.biz = workTask.getBiz();
    }
}