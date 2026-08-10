package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import lombok.extern.slf4j.Slf4j;

@Slf4j
public class CozeReader extends ProviderReader<CozeRequest> {

    public CozeReader(ProviderReaderConfig<CozeRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }
}