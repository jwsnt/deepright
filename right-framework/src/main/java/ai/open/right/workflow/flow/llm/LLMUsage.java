package ai.open.right.workflow.flow.llm;

public interface LLMUsage {

    public Integer getThinking();

    public Integer getCache();

    public Integer getTotal();

    public Integer getInput();

    public void addUsage(LLMUsage usage);
}
