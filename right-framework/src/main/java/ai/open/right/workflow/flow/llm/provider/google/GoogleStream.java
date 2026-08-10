package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.media.MediaInlineData;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.List;
import java.util.Map;

@Slf4j
public class GoogleStream extends ProviderStream<GoogleRequest> {

    protected Boolean forceNotifierProcess = false;

    protected String thoughtSignature;

    public GoogleStream(ProviderStreamConfig<GoogleRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected Boolean stream(String source) throws Exception {
        boolean finished = false;
        Map<String, Object> body = JsonUtils.read(source, Map.class);
        Assert.notEmpty(body, "Body can not be empty");
        String finishReason = MapUtils.getString(body, "finishReason");
        Assert.isTrue(StringUtils.isEmpty(finishReason), "Vertex encountered an exception: " + finishReason);
        List<Map<String, Object>> candidates = List.class.cast(body.get("candidates"));
        WorkflowException.checkCondition(CollectionUtils.isEmpty(candidates), "Candidates can not be empty", ProtocolCode.C914);
        Boolean totalFinish = false;
        for (Map<String, Object> candidate : candidates) {
            finishReason = MapUtils.getString(candidate, "finishReason");
            if (StringUtils.equalsIgnoreCase(finishReason, "STOP")) {
                // 标记STOP
                finished = true;
            } else if (!StringUtils.isEmpty(finishReason)) {
                // 记录用量
                this.tokenStatistic(body);
                // 非STOP异常
                throw new WorkflowException(StringUtils.defaultIfBlank(MapUtils.getString(candidate, "finishMessage"), finishReason), ProtocolCode.C914);
            }
            Map content = MapUtils.getMap(candidate, "content");
            // 如果Content为空则读取finishReason + finishMessage
            if (!CollectionUtils.isEmpty(content)) {
                String role = MapUtils.getString(content, "role");
                if ("model".equalsIgnoreCase(role)) {
                    List<Map<String, Object>> parts = List.class.cast(content.get("parts"));
                    Assert.notEmpty(parts, "Parts can not be empty");
                    for (Map<String, Object> part : parts) {
                        // 思考链路
                        this.addReason(part, finished);
                        if (!CollectionUtils.isEmpty(MapUtils.getMap(part, "functionCall"))) {
                            // AsyncFunction call redirect to atonce(and return)
                            this.forceNotifierProcess = true;
                            return this.atonce(source);
                        }
                        // 二进制文件处理
                        Map<String, Object> inline = MapUtils.getMap(part, "inlineData");
                        if (inline != null) {
                            this.addInlineData(inline);
                        }
                        // Markdown格式中，content可能为有意义的空字符
                        this.addContent(MapUtils.getString(part, "text", ""), false);
                    }
                }
            }
            // 标记流
            this.notify(this.seqid++, false);
            totalFinish = (totalFinish || finished);
            if (totalFinish) {
                this.responseCheck();
                if (this.forceNotifierProcess) {
                    this.forceNotifierProcess = false;
                    this.notifyProcess();
                } else {
                    // 结束并推送处理（如果存在FunCall将转交AtOnce）
                    this.notifySegment();
                }
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("Google parse success, finished={}, source={}", finished, source);
        }
        // 记录用量
        this.tokenStatistic(body);
        return totalFinish;
    }

    @Override
    protected Boolean atonce(String source) throws Exception {
        boolean finished = false;
        Map<String, Object> body = JsonUtils.read(source, Map.class);
        Assert.notEmpty(body, "Body can not be empty");
        String finishReason = MapUtils.getString(body, "finishReason");
        Assert.isTrue(StringUtils.isEmpty(finishReason), "Vertex encountered an exception: " + finishReason);
        List<Map<String, Object>> candidates = List.class.cast(body.get("candidates"));
        WorkflowException.checkCondition(CollectionUtils.isEmpty(candidates), "Candidates can not be empty", ProtocolCode.C914);
        for (Map<String, Object> candidate : candidates) {
            finishReason = MapUtils.getString(candidate, "finishReason");
            if (StringUtils.equalsIgnoreCase(finishReason, "STOP")) {
                // 标记STOP
                finished = true;
            } else if (!StringUtils.isEmpty(finishReason)) {
                // 记录用量
                this.tokenStatistic(body);
                // 非STOP异常
                throw new WorkflowException(StringUtils.defaultIfBlank(MapUtils.getString(candidate, "finishMessage"), finishReason), ProtocolCode.C914);
            }
            Map content = MapUtils.getMap(candidate, "content");
            // 如果Content为空则读取finishReason + finishMessage
            if (!CollectionUtils.isEmpty(content)) {
                String role = MapUtils.getString(content, "role");
                if ("model".equalsIgnoreCase(role)) {
                    List<Map<String, Object>> parts = List.class.cast(content.get("parts"));
                    Assert.notEmpty(parts, "Parts can not be empty");
                    for (Map<String, Object> part : parts) {
                        // 思考链路
                        this.addReason(part, finished);
                        Map<String, Object> functionCall = MapUtils.getMap(part, "functionCall");
                        if (!CollectionUtils.isEmpty(functionCall)) {
                            // 处理Fun Call
                            this.thoughtSignature = StringUtils.defaultIfEmpty(MapUtils.getString(part, "thoughtSignature"), this.thoughtSignature);
                            part.putIfAbsent("thoughtSignature", this.thoughtSignature);
                            this.addFunRequest(ProviderFunCallRequest.builder()
                                    .name(MapUtils.getString(functionCall, "name"))
                                    .args(MapUtils.getMap(functionCall, "args"))
                                    .model(this.request.getModel())
                                    .api(this.request.getApi())
                                    .refer(part)
                                    .build());
                        }
                        // 二进制文件处理
                        Map<String, Object> inline = MapUtils.getMap(part, "inlineData");
                        if (inline != null) {
                            this.addInlineData(inline);
                        }
                        // Markdown格式中，content可能为有意义的空字符
                        this.addContent(MapUtils.getString(part, "text", ""), false);
                    }
                }
            }
        }
        if (log.isDebugEnabled()) {
            log.debug("Google parse success, finished={}, source={}", finished, source);
        }
        if (finished) {
            // 结束并推送处理
            this.forceNotifierProcess = false;
            this.notifyProcess();
        }
        this.responseCheck();
        // 记录用量
        this.tokenStatistic(body);
        return finished;
    }

    protected void addInlineData(Map<String, Object> data) throws Exception {
        Assert.notNull(this.mediaInlineService, "The media inline service can not be empty, please config `media.enable`");
        this.addContent(this.mediaInlineService.write(MediaInlineData.builder()
                .mediaType(MapUtils.getString(data, "mimeType"))
                .data(MapUtils.getString(data, "data"))
                .build(), this.request.getMessage()), false);
    }

    @Override
    protected void tokenStatistic(Map<String, Object> body) throws Exception {
        // totalTokenCount：本次请求的总Token消耗，即输入与输出的总和。
        // promptTokenCount：输入的提示词长度，包含了用户指令、上下文和多轮对话历史
        // candidatesTokenCount：模型最终生成并返回给的回答长度
        // cachedContentTokenCount：命中了Context Caching的Token数，通常计费更低且响应更快
        // thoughtsTokenCount：模型生成答案前的思考过程长度（推理模型特有，包含在输出计数中）
        // {"usageMetadata":{"promptTokenCount":7512,"candidatesTokenCount":157,"totalTokenCount":8129,"cachedContentTokenCount":3573,"trafficType":"ON_DEMAND","promptTokensDetails":[{"modality":"TEXT","tokenCount":7512}],"cacheTokensDetails":[{"modality":"TEXT","tokenCount":3573}],"candidatesTokensDetails":[{"modality":"TEXT","tokenCount":157}],"thoughtsTokenCount":460}}
        Map<String, Object> usage = MapUtils.getMap(body, "usageMetadata");
        Integer cache = MapUtils.getInteger(usage, "cachedContentTokenCount", 0);
        Integer thinking = MapUtils.getInteger(usage, "thoughtsTokenCount", 0);
        Integer input = MapUtils.getInteger(usage, "promptTokenCount", 0);
        Integer total = MapUtils.getInteger(usage, "totalTokenCount", 0);
        if (total != 0 || input != 0 || cache != 0 || thinking != 0) {
            // 不为0时记录
            TokenData tokenData = TokenData.builder()
                    .thinking(thinking)
                    .cache(cache)
                    .input(input)
                    .total(total)
                    .build();
            this.tokenStatistic.stat(this.request, tokenData);
            this.segment.setUsage(new SegmentUsage(tokenData));
            if (log.isInfoEnabled()) {
                log.info("The token statistic={}, total={}, cache={}", this.tokenStatistic, total, cache);
            }
        }
    }
}
