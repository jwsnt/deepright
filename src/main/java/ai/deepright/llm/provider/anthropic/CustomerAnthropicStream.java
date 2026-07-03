package ai.deepright.llm.provider.anthropic;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicStream;
import ai.deepright.llm.provider.RequestFunCallStore;

public class CustomerAnthropicStream extends AnthropicStream {

    public CustomerAnthropicStream(ProviderStreamConfig<AnthropicRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
        RequestFunCallStore.shouldStoreFunCallData(this.request, request, response, this.historyStore, this.namesService);
    }
}
