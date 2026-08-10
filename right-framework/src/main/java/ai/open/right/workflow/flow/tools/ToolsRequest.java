package ai.open.right.workflow.flow.tools;

import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;

@Setter
@Getter
public class ToolsRequest implements ToolsStructure {

    protected Map<String, Object> metadata;

    protected UserContext userContext;

    protected String conversation;

    protected String workflow;

    protected Long timestamp;

    protected String trace;

    protected Object query;

    protected String chat;

    protected String biz;

    public ToolsRequest(WorkflowTask workTask, Object query) {
        this.conversation = workTask.getConversation();
        this.userContext = workTask.getUserContext();
        this.timestamp = workTask.getCreated();
        this.metadata = workTask.getMetadata();
        this.workflow = workTask.getWorkflow();
        this.trace = workTask.getTrace();
        this.chat = workTask.getChat();
        this.biz = workTask.getBiz();
        this.query = query;
    }

    public ToolsRequest() {
    }
}
