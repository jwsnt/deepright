package ai.deepright.llm.provider.deepseek;

import ai.deepright.llm.provider.RequestThinkingUtils;
import ai.deepright.llm.provider.openai.CustomerOpenAiStreamFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;

import java.util.Map;

public class CustomerDeepseekStream extends CustomerOpenAiStreamFunCall {

    public CustomerDeepseekStream(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
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
