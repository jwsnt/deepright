package ai.open.right.workflow.flow.llm;

import ai.open.right.workflow.flow.llm.token.TokenData;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import org.apache.commons.collections.MapUtils;

import java.util.Map;

@Getter
// {"prompt_tokens":5000,"completion_tokens":100,"total_tokens":5100,"prompt_tokens_details":{"cached_tokens":3840},"completion_tokens_details":{"reasoning_tokens":0,"accepted_prediction_tokens":0,"rejected_prediction_tokens":0}}
// prompt_tokens：输入消耗，包含发送的指令、文档内容及关联的上下文历史
// completion_tokens：输出消耗，模型实际生成并返回给的回答文本长度
// total_tokens：计费总量，即prompt_tokens与completion_tokens求和
// reasoning_tokens：推理消耗产生的Token（包含在输出计数中）
// cached_tokens：缓存命中
// {"prompt_tokens":5000,"completion_tokens":100,"total_tokens":5100,"prompt_tokens_details":{"cached_tokens":3840},"completion_tokens_details":{"reasoning_tokens":0,"accepted_prediction_tokens":0,"rejected_prediction_tokens":0}}
public class SegmentUsage implements LLMUsage {

    @JsonProperty("completion_tokens_details")
    protected Map<String, Object> reasoning;

    @JsonProperty("prompt_tokens_details")
    protected Map<String, Object> details;

    @JsonProperty("prompt_tokens")
    protected Integer input = 0;

    @JsonProperty("total_tokens")
    protected Integer total = 0;

    public SegmentUsage(TokenData tokenData) {
        this.reasoning = ImmutableMap.of("reasoning_tokens", tokenData.getThinking());
        this.details = ImmutableMap.of("cached_tokens", tokenData.getCache());
        this.total = tokenData.getTotal();
        this.input = tokenData.getInput();
    }

    public SegmentUsage() {
        this.reasoning = ImmutableMap.of("reasoning_tokens", 0);
        this.details = ImmutableMap.of("cached_tokens", 0);
        this.total = 0;
        this.input = 0;
    }

    @Override
    public Integer getThinking() {
        return MapUtils.getInteger(this.reasoning, "reasoning_tokens", 0);
    }

    @Override
    public Integer getCache() {
        return MapUtils.getInteger(this.details, "cached_tokens", 0);
    }

    @Override
    public void addUsage(LLMUsage usage) {
        Integer targetReasoning = usage.getThinking();
        Integer targetCache = usage.getCache();
        this.reasoning = ImmutableMap.of("reasoning_tokens", this.getThinking() + (targetReasoning != null ? targetReasoning : 0));
        this.details = ImmutableMap.of("cached_tokens", this.getCache() + (targetCache != null ? targetCache : 0));
        this.total += usage.getTotal();
        this.input += usage.getInput();
    }
}
