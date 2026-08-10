package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;

public class AnthropicReader extends ProviderReader<AnthropicRequest> {

    public AnthropicReader(ProviderReaderConfig<AnthropicRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }
}