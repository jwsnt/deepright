package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.llm.Segment;

public interface ProviderStorePolicy {

    public Boolean shouldStoreFunCallData(ProviderRequest request, ProviderFunCallRequest funCallRequest, ProviderFunCallResponse funCallResponse) throws Exception;

    public Boolean shouldStoreConversation(ProviderRequest request, Segment segment, String content) throws Exception;
}