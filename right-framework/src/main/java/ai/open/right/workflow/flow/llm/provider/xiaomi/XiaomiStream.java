package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiStreamFunCall;
import org.apache.commons.lang3.StringUtils;

public class XiaomiStream extends OpenAiStreamFunCall {

    public XiaomiStream(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    public Boolean stream(String source) throws Exception {
        return !StringUtils.startsWithIgnoreCase(source, ": PROCESSING") && super.stream(source);
    }
}
