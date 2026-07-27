package ai.deepright.llm.provider.xiaomi;

import ai.deepright.llm.provider.RequestFunCallStore;
import ai.deepright.llm.provider.RequestThinkingUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.xiaomi.XiaomiStream;

import java.util.Map;

public class CustomerXiaomiStream extends XiaomiStream {

    public CustomerXiaomiStream(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected void storeFunCallData(ProviderFunCallRequest request, ProviderFunCallResponse response) throws Exception {
        RequestFunCallStore.shouldStoreFunCallData(this.request, request, response, this.historyStore, this.namesService);
    }

    @Override
    // 子类覆盖
    protected void addReason(Map<String, Object> message, Boolean finished) throws Exception {
        super.addReason(message, finished);
        if (finished) {
            RequestThinkingUtils.notifyMessage(this.notifierService, this.request.getMessage(), this.reasoning);
        }
    }
}