package ai.open.right.workflow.flow.llm.provider.reason;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;

public interface ProviderReason {

    public String reason(ProviderRequest request, String message, Boolean finished, Integer index) throws Exception;
}
