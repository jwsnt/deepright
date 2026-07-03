package ai.deepright.llm.provider.openai;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderCallback;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiReader;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.deepright.llm.RetryCallback;
import ai.deepright.llm.RetryUtils;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.protocol.HttpContext;

@Slf4j
public class CustomerOpenAiReader extends OpenAiReader {

    public CustomerOpenAiReader(ProviderReaderConfig<OpenAiRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }

    @Override
    protected ProviderReaderCallback buildReaderCallback(ProviderReaderConfig<OpenAiRequest> providerReaderConfig) throws Exception {
        return new RetryCallback((ProviderReaderConfig<ProviderRequest>) (ProviderReaderConfig<?>) providerReaderConfig, this.messageQueue, this.request, this.request.getMessage());
    }

    @Override
    protected Void buildResult(HttpContext httpContext) {
        try {
            return super.buildResult(httpContext);
        } finally {
            try {
                // 清除重试标记
                RetryUtils.clean(this.request.getMessage());
            } catch (Exception e) {
                // 不可以抛出异常
                WorkflowException.dolog(e);
            }
        }
    }
}
