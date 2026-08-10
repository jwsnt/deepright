package ai.open.right.workflow.flow.llm;

import java.util.List;
import java.util.Map;

public interface LLMFunCallData {

    public List<LLMFunCallResponse> getResponses();

    public List<LLMFunCallRequest> getRequests();

    public Map<String, Object> getMetadata();

    public <T> T getMetadata(String key, Class<T> clazz) throws Exception;

    public void putMetadata(String key, Object value);

    public void setResponses(List<LLMFunCallResponse> funCallResponses);

    public void setRequests(List<LLMFunCallRequest> funCallRequests);
}
