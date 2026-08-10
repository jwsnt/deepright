package ai.open.right.workflow.flow.llm.provider.anthropic;

import java.util.Set;

public interface AnthropicMedia {

    // Image/Document
    public String getType(String type) throws Exception;

    public Set<String> getMimes() throws Exception;
}
