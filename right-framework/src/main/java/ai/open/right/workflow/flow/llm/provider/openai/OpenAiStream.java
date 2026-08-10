package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.SegmentUsage;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.token.TokenData;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;

@Slf4j
public class OpenAiStream extends ProviderStream<OpenAiRequest> {

    public OpenAiStream(ProviderStreamConfig<OpenAiRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    public void callback(String message) throws Exception {
        if (!StringUtils.startsWithIgnoreCase(ProviderReader.DONE, message)) {
            super.callback(message);
        } else if (log.isDebugEnabled()) {
            log.debug("OpenAiStream received the [done]={}", message);
        }
    }

    @Override
    public Boolean stream(String source) throws Exception {
        Assert.isTrue(StringUtils.startsWithIgnoreCase(source, "data:"), "Invalid data");
        Map<String, Object> body = JsonUtils.read(source.replaceFirst("data:", ""), Map.class);
        Assert.notNull(body, "Body can not be empty");
        Assert.notNull(body.get("id"), "Body id can not be empty");
        List<Map<String, Object>> choices = List.class.cast(body.get("choices"));
        // 使用NotNull而不是Empty
        Assert.notNull(choices, "Choices can not be empty");
        boolean totalFinish = false;
        for (Map<String, Object> choice : choices) {
            Object finishReason = choice.get("finish_reason");
            boolean finish = finishReason != null && ("stop".equalsIgnoreCase(finishReason.toString()) || "tool_calls".equalsIgnoreCase(finishReason.toString()));
            Map<String, Object> delta = MapUtils.getMap(choice, "delta");
            if (!MapUtils.isEmpty(delta)) {
                // Markdown格式中，content可能为有意义的空字符
                this.addContent(MapUtils.getString(delta, "content", ""), finish);
                // 用于子类覆盖
                this.addReason(delta, false);
                this.addMetadata(delta);
                // 处理Fun Call
                this.addFunRequest(delta, List.class.cast(delta.get("tool_calls")));
            }
            this.notify(this.seqid++, false);
            totalFinish = (totalFinish || finish);
            if (totalFinish) {
                this.addReason(null, true);
                this.responseCheck();
                // 提交FunCall&记忆
                this.notifyProcess();
            }
        }
        // 用量统计
        this.tokenStatistic(body);
        return totalFinish;
    }

    @Override
    public Boolean atonce(String source) throws Exception {
        Map<String, Object> body = JsonUtils.read(source, Map.class);
        Assert.notNull(body, "Body can not be empty");
        Assert.notNull(body.get("id"), "Body id can not be empty");
        List<Map<String, String>> messages = List.class.cast(body.get("choices"));
        Assert.notEmpty(messages, "Choices can not be empty");
        for (Map<String, String> each : messages) {
            Map<String, Object> message = MapUtils.getMap(each, "message");
            Assert.notEmpty(message, "Message can not be empty");
            if ("assistant".equals(message.get("role"))) {
                // Markdown格式中，content可能为有意义的空字符
                this.addContent(MapUtils.getString(message, "content", ""), true);
                // 用于子类覆盖
                this.addReason(message, false);
                this.addMetadata(message);
                this.addFunRequest(message, List.class.cast(message.get("tool_calls")));
                break;
            }
        }
        this.addReason(null, true);
        this.responseCheck();
        this.notifyProcess();
        // 用量统计
        this.tokenStatistic(body);
        return true;
    }

    protected void addFunRequest(Map<String, Object> message, List<Map<String, Object>> functionCalls) throws Exception {
        if (CollectionUtils.isEmpty(functionCalls)) {
            return;
        }
        for (Map<String, Object> functionCall : functionCalls) {
            Map<String, Object> function = MapUtils.getMap(functionCall, "function");
            Assert.notEmpty(function, "Function can not be empty");
            String id = MapUtils.getString(functionCall, "id");
            String name = MapUtils.getString(function, "name");
            Assert.hasText(name, "Function's name can not be empty");
            Assert.hasText(id, "Function's id can not be empty");
            this.addFunRequest(ProviderFunCallRequest.builder()
                    .reason(!StringUtils.isEmpty(this.reasoning) ? this.reasoning.toString() : "")
                    .args(function.get("arguments"))
                    .model(this.request.getModel())
                    .api(this.request.getApi())
                    .refer(functionCall)
                    .name(name)
                    .id(id)
                    .build());
        }
    }

    @Override
    // 子类覆盖
    protected void addReason(Map<String, Object> message, Boolean finished) throws Exception {
        String reasoning = MapUtils.getString(message, "reasoning_content");
        if (!StringUtils.isEmpty(reasoning)) {
            // 追加Reasoning
            this.reasoning = this.reasoning != null ? this.reasoning : new StringBuffer();
            this.reasoning.append(reasoning);
            if (this.request.getPrintReason()) {
                // 对于DeepSeek, 输出Reason给终端会（默认）记录在多轮会话中，产生双倍容量（reasoning_content本身会记录属性）
                // 可通过子类覆盖自定义实现
                this.addContent(this.providerReason.reason(this.request, reasoning, finished, this.reasonIdx++), false);
            }
        }
    }

    // 子类覆盖
    protected void addMetadata(Map<String, Object> message) throws Exception {
    }

    @Override
    protected void tokenStatistic(Map<String, Object> body) throws Exception {
        // prompt_tokens：输入消耗，包含发送的指令、文档内容及关联的上下文历史
        // completion_tokens：输出消耗，模型实际生成并返回给的回答文本长度
        // total_tokens：计费总量，即prompt_tokens与completion_tokens求和
        // reasoning_tokens：推理消耗产生的Token（包含在输出计数中）
        // cached_tokens：缓存命中
        // {"prompt_tokens":5000,"completion_tokens":100,"total_tokens":5100,"prompt_tokens_details":{"cached_tokens":3840},"completion_tokens_details":{"reasoning_tokens":0,"accepted_prediction_tokens":0,"rejected_prediction_tokens":0}}
        Map<String, Object> usage = MapUtils.getMap(body, "usage");
        Integer thinking = MapUtils.getInteger(MapUtils.getMap(usage, "completion_tokens_details"), "reasoning_tokens", 0);
        Integer cache = MapUtils.getInteger(MapUtils.getMap(usage, "prompt_tokens_details"), "cached_tokens", 0);
        Integer input = MapUtils.getInteger(usage, "prompt_tokens", 0);
        Integer total = MapUtils.getInteger(usage, "total_tokens", 0);
        if (cache != 0 || total != 0 || input != 0) {
            // 任一不为0时记录
            TokenData tokenData = TokenData.builder()
                    .thinking(thinking)
                    .cache(cache)
                    .total(total)
                    .input(input)
                    .build();
            this.tokenStatistic.stat(this.request, tokenData);
            this.segment.setUsage(new SegmentUsage(tokenData));
            if (log.isInfoEnabled()) {
                log.info("The token statistic={}, total={}, cache={}", this.tokenStatistic, total, cache);
            }
        }
    }
}
