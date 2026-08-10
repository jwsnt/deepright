package ai.open.right.workflow.flow.llm.provider.seedream;

import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;

public class SeedreamReader extends ProviderReader<SeedreamRequest> {

    public SeedreamReader(ProviderReaderConfig<SeedreamRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }
}