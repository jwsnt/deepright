package ai.deepright.llm.provider.openai;

import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStreamFunCall;
import ai.deepright.llm.provider.RequestFunCallStore;

public class CustomerOpenAiStreamFunCall extends OpenAiStreamFunCall {

    public CustomerOpenAiStreamFunCall(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
        RequestFunCallStore.shouldStoreFunCallData(this.request, request, response, this.historyStore, this.namesService);
    }
}
