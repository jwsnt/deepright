package ai.deepright.llm.optimize.compressor;

import ai.deepright.llm.optimize.FunCallCompressor;
import ai.deepright.module.HttpProtocol;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;

import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;
import java.util.Map;

@Getter
@Setter
@Slf4j
abstract public class StoreCompressor implements FunCallCompressor {

    protected HttpProtocol httpProtocol;

    protected Integer prefixReserve;

    protected Integer suffixReserve;

    protected SysStore sysStore;

    // 压缩单条FunCall的阈值（50k）
    protected Integer oversize;

    @Override
    public void compress(ProviderRequest providerRequest, LLMFunCallResponse funCallResponse) throws Exception {
        if (this.shouldCompress(funCallResponse)) {
            // URL压缩在Query中
            String url = this.buildUrl(providerRequest, this.sysStore.store(funCallResponse.getResponse().getBytes(StandardCharsets.UTF_8), ".json", providerRequest.getMessage()));
            funCallResponse.setResponse(this.buildRecallAnswer(funCallResponse, url));
            funCallResponse.putMetadata(FunCallCompressor.FLAG, true);
            if (log.isWarnEnabled()) {
                log.warn("The response will be compressed, device={}", providerRequest.getMessage().getDevice());
            }
        }
    }

    @Override
    public void compress(ProviderRequest providerRequest, LLMFunCallRequest funCallRequest) throws Exception {
        if (this.shouldCompress(funCallRequest)) {
            String original = JsonUtils.write(Map.class.cast(funCallRequest.getArgs()));
            // URL压缩在属性中
            String url = this.buildUrl(providerRequest, this.sysStore.store(original.getBytes(StandardCharsets.UTF_8), ".json", providerRequest.getMessage()));
            funCallRequest.setArgs(ImmutableMap.of("the_original_digest", this.buildRecallQuery(providerRequest, original), "the_complete_content", url));
            funCallRequest.putMetadata(FunCallCompressor.FLAG, true);
            if (log.isWarnEnabled()) {
                log.warn("The request will be compressed, device={}", providerRequest.getMessage().getDevice());
            }
        }
    }

    // 构建原始请求摘要
    protected String buildRecallQuery(ProviderRequest providerRequest, String original) throws Exception {
        StringBuffer buffer = new StringBuffer();
        buffer.append(this.buildPrefixReserve(original)).append(StringUtils.repeat(" ", 3));
        buffer.append("... [The complete content is saved at `the_complete_content` field] ... If recall is needed, please use `curl` via tool `[cli]`");
        buffer.append(StringUtils.repeat(" ", 3)).append(this.buildSuffixReserve(original));
        return buffer.toString();
    }

    // 构建原始响应摘要
    protected String buildRecallAnswer(LLMFunCallResponse funCallResponse, String url) throws Exception {
        StringBuffer buffer = new StringBuffer();
        buffer.append(this.buildPrefixReserve(funCallResponse.getResponse())).append(StringUtils.repeat(" ", 3));
        buffer.append("... [The remaining content is saved at `").append(url).append("`] ... If recall is needed, please use `curl` via tool `[cli]`");
        buffer.append(StringUtils.repeat(" ", 3)).append(this.buildSuffixReserve(funCallResponse.getResponse()));
        return buffer.toString();
    }

    // 构建召回URL
    protected String buildUrl(ProviderRequest providerRequest, String url) throws Exception {
        return !MediaTransferUtils.isNetwork(url) ? this.httpProtocol.dataHost() + Paths.get(url).getFileName() : url;
    }

    // 未压缩且超过指定大小
    protected Boolean shouldCompress(LLMFunCallResponse funCallResponse) throws Exception {
        return (!MapUtils.getBooleanValue(funCallResponse.getMetadata(), FunCallCompressor.FLAG, false)) && (BytesUtils.utf8Bytes(funCallResponse.getResponse()) > this.oversize);
    }

    // 未压缩且超过指定大小
    protected Boolean shouldCompress(LLMFunCallRequest funCallRequest) throws Exception {
        return (!MapUtils.getBooleanValue(funCallRequest.getMetadata(), StoreCompressor.FLAG, false)) && (BytesUtils.utf8Bytes(JsonUtils.write(funCallRequest.getRefer())) > this.oversize);
    }

    protected String buildSuffixReserve(String content) {
        return StringUtils.right(content, this.suffixReserve);
    }

    protected String buildPrefixReserve(String content) {
        return StringUtils.left(content, this.prefixReserve);
    }

    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected HttpProtocol httpProtocol;

        @Autowired
        protected SysStore sysStore;

        @Value("${optimize.oversize.funcall.each.prefix.reserve:50}")
        protected Integer prefixReserve;

        @Value("${optimize.oversize.funcall.each.suffix.reserve:0}")
        protected Integer suffixReserve;

        // 51,200字节，50k
        @Value("${optimize.oversize.funcall.each:51200}")
        protected Integer oversize;
    }
}
