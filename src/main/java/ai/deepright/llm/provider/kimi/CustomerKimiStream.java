package ai.deepright.llm.provider.kimi;

import ai.deepright.llm.provider.RequestThinkingUtils;
import ai.deepright.llm.provider.openai.CustomerOpenAiStreamFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.apache.commons.collections.MapUtils;

import java.util.Map;

public class CustomerKimiStream extends CustomerOpenAiStreamFunCall {

    public CustomerKimiStream(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    // 子类覆盖
    protected void addReason(Map<String, Object> message, Boolean finished) throws Exception {
        super.addReason(message, finished);
        RequestThinkingUtils.notifyMessage(this.notifierService, this.request.getMessage(), MapUtils.getString(message, "reasoning_content"));
    }
}
