package ai.open.right.workflow.flow.llm;

public interface LLMCallback {

    public void callback(String message) throws Exception;
}