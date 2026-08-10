package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;

public class OpenAiReader extends ProviderReader<OpenAiRequest> {

    public OpenAiReader(ProviderReaderConfig<OpenAiRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }
}