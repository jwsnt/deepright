package ai.open.right.workflow.flow.llm;

import java.util.Map;

public interface LLMFunCallRequest {

    public Map<String, Object> getMetadata();

    public Map<String, Object> getRefer();

    public Long getCreated();

    public String getReason();

    public Object getArgs();

    public String getName();

    public String getId();

    public void setMetadata(Map<String, Object> metadata);

    public void putMetadata(String key, Object value);

    public void setRefer(Map<String, Object> refer);

    public void setName(String name);

    public void setArgs(Object args);

    public void setId(String id);

    public Boolean isValid();
}
