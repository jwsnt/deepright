package ai.open.right.workflow.flow.tools;

import ai.open.right.context.UserContext;

import java.util.Map;

public interface ToolsStructure {

    public Map<String, Object> getMetadata();

    public UserContext getUserContext();

    public String getConversation();

    public String getWorkflow();

    public Long getTimestamp();

    public String getTrace();

    public Object getQuery();

    public String getChat();

    public String getBiz();
}
