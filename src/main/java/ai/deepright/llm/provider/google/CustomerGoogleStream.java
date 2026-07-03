package ai.deepright.llm.provider.google;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleStream;
import ai.deepright.llm.provider.RequestFunCallStore;

public class CustomerGoogleStream extends GoogleStream {

    public CustomerGoogleStream(ProviderStreamConfig<GoogleRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
        RequestFunCallStore.shouldStoreFunCallData(this.request, request, response, this.historyStore, this.namesService);
    }
}
