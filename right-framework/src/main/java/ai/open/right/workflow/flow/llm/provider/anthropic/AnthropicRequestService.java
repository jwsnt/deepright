package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class AnthropicRequestService extends ProviderRequestService<AnthropicRequest> implements ProviderRequestModel {

    public static final Map<String, Object> THINK_CONFIG = ImmutableMap.of("type", "adaptive", "display", "omitted");

    public static final Map<String, Object> CACHE_CONTROL = ImmutableMap.of("type", "ephemeral");

    public static final String NAME = "AnthropicRequestService";

    protected String reasoningEffort;

    protected String model;

    protected String token;

    @Override
    protected AnthropicRequest build() throws Exception {
        return new AnthropicRequest();
    }

    @Override
    protected void request(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setApi(ProviderRequest.REQUEST_ANTHROPIC);
        this.reasoning(request, llmConfig, llmQuery);
        this.extra(request, llmConfig, llmQuery);
    }

    @Override
    protected String defToken(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__token"), this.token);
    }

    @Override
    protected String defModel(WorkflowTask workTask) throws Exception {
        String model = StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__model"), this.getModel(workTask));
        Assert.hasText(model, "The model can not be empty");
        return model;
    }

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    protected void reasoning(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://platform.claude.com/docs/en/build-with-claude/extended-thinking#how-to-use-extended-thinking
        // 同标准协议thinking={"type": "enabled"}
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        // https://platform.claude.com/docs/en/build-with-claude/effort
        // response = client.messages.create(
        //    model="claude-opus-4-8",
        //    max_tokens=4096,
        //    messages=[
        //        {
        //            "role": "user",
        //            "content": "Analyze the trade-offs between microservices and monolithic architectures",
        //        }
        //    ],
        //    output_config={"effort": "medium"},
        //)
        if (!MapUtils.isEmpty(thinking) && StringUtils.equalsAnyIgnoreCase(MapUtils.getString(thinking, "type"), "enabled", "adaptive")) {
            request.setThinking(AnthropicRequestService.THINK_CONFIG);
            // output_config.effort可独立于thinking使用，但此处统一
            this.reasoningEffort(request, llmConfig, llmQuery);
        } else {
            request.setThinking(thinking);
        }
    }

    protected void reasoningEffort(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        Map<String, Object> responseFormat = request.getResponseFormat();
        responseFormat = responseFormat != null ? new HashMap<String, Object>(responseFormat) : new HashMap<String, Object>();
        String reasoning = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
        reasoning = !StringUtils.isEmpty(reasoning) ? reasoning : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
        reasoning = !StringUtils.isEmpty(reasoning) ? reasoning : this.reasoningEffort;
        responseFormat.put("effort", reasoning);
        request.setResponseFormat(responseFormat);
    }

    protected void extra(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // 默认开启，"cache_control": { "type": "ephemeral" }
        // 子类可覆盖
        request.setCacheControl(AnthropicRequestService.CACHE_CONTROL);
    }

    @ConditionalOnProperty(name = "anthropic.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${anthropic.model.reasoningEffort:low}")
        protected String reasoningEffort;

        @Value("${anthropic.model:claude-sonnet-5}")
        protected String model;

        @Value("${anthropic.token:}")
        protected String token;

        @Bean(name = AnthropicRequestService.NAME)
        @ConditionalOnMissingBean(name = AnthropicRequestService.NAME)
        public AnthropicRequestService anthropicRequestService() throws Exception {
            AnthropicRequestService anthropicRequestService = new AnthropicRequestService();
            BeanUtils.copyProperties(this, anthropicRequestService);
            log.info("AnthropicRequestService inited. model={}, token={}, timeout={}", anthropicRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(anthropicRequestService.getToken())), anthropicRequestService.getFunCallTimeout());
            return anthropicRequestService;
        }
    }
}
