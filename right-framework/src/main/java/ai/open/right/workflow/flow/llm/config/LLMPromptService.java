package ai.open.right.workflow.flow.llm.config;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;

public interface LLMPromptService {

    public String prompt(ProviderRequest request, LLMConfig config, LLMQuery query) throws Exception;
}
