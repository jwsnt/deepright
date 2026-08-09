package ai.deepright.llm.optimize;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;

public interface FunCallCompressor {

    public static final String FLAG = "__compress__";

    public void compress(ProviderRequest request, LLMFunCallResponse funCallResponse) throws Exception;

    public void compress(ProviderRequest request, LLMFunCallRequest funCallRequest) throws Exception;
}
