package ai.deepright.llm.provider.minimax;

import ai.deepright.llm.provider.RequestThinkingUtils;
import ai.deepright.llm.provider.anthropic.CustomerAnthropicStream;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;

import java.util.Map;

public class CustomerMiniMaxStream extends CustomerAnthropicStream {

    public CustomerMiniMaxStream(ProviderStreamConfig<AnthropicRequest> providerRequestConfig) throws Exception {
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
