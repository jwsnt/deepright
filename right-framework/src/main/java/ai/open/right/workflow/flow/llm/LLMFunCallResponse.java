package ai.open.right.workflow.flow.llm;

import java.util.Map;

public interface LLMFunCallResponse {

    public Map<String, Object> getMetadata();

    public String getResponse();

    public String getName();

    public String getId();

    public Long getCreated();

    public void setMetadata(Map<String, Object> metadata);

    public void putMetadata(String key, Object value);

    public void setResponse(String response);

    public void setName(String name);

    public void setId(String id);

    public Boolean isValid();
}
