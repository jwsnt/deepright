package ai.open.right.workflow.flow.llm.provider.openai;

import java.util.Set;

public interface OpenAiMedia {

    public String getPrefix(String type) throws Exception;

    public String getKeyUrl(String type) throws Exception;

    public Set<String> getMimes() throws Exception;
}
