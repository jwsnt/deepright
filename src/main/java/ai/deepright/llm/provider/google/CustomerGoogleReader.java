package ai.deepright.llm.provider.google;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderCallback;
import ai.open.right.workflow.flow.llm.provider.ProviderReaderConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.google.GoogleReader;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.deepright.llm.RetryCallback;
import ai.deepright.llm.RetryUtils;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.protocol.HttpContext;

@Slf4j
public class CustomerGoogleReader extends GoogleReader {

    public CustomerGoogleReader(ProviderReaderConfig<GoogleRequest> providerReaderConfig) throws Exception {
        super(providerReaderConfig);
    }

    @Override
    protected ProviderReaderCallback buildReaderCallback(ProviderReaderConfig<GoogleRequest> providerReaderConfig) throws Exception {
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
