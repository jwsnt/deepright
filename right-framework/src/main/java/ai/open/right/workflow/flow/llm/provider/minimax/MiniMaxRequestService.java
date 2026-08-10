package ai.open.right.workflow.flow.llm.provider.minimax;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequest;
import ai.open.right.workflow.flow.llm.provider.anthropic.AnthropicRequestService;
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

import java.util.Map;

@Setter
@Getter
@Slf4j
public class MiniMaxRequestService extends AnthropicRequestService implements ProviderRequestModel {

    public static final Map<String, Object> THINK_CONFIG = ImmutableMap.of("type", "adaptive");

    public static final String NAME = "MiniMaxRequestService";

    protected String model;

    protected String token;

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
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
    protected void reasoning(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://platform.minimaxi.com/docs/api-reference/text-openai-api#thinking-%E6%8E%A7%E5%88%B6
        // 不支持reasoning_effort
        // response = client.chat.completions.create(
        // model="MiniMax-M3",
        // messages=[{"role": "user", "content": "Hi, how are you?"}],
        // extra_body={
        //  "thinking": {"type": "adaptive"},
        // },
        // )
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        if (!MapUtils.isEmpty(thinking)) {
            request.setExtra(ProviderRequestService.KEY_THINKING, StringUtils.equalsIgnoreCase(MapUtils.getString(thinking, "type"), "disabled") ? thinking : ImmutableMap.of("type", "adaptive"));
        }
    }

    @Override
    protected void extra(AnthropicRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    @ConditionalOnProperty(name = "minimax.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${minimax.model:MiniMax-M3}")
        protected String model;

        @Value("${minimax.token:}")
        protected String token;

        @Bean(name = MiniMaxRequestService.NAME)
        @ConditionalOnMissingBean(name = MiniMaxRequestService.NAME)
        public MiniMaxRequestService miniMaxRequestService() throws Exception {
            MiniMaxRequestService miniMaxRequestService = new MiniMaxRequestService();
            BeanUtils.copyProperties(this, miniMaxRequestService);
            log.info("MiniMaxRequestService inited. model={}, token={}, timeout={}", miniMaxRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(miniMaxRequestService.getToken())), miniMaxRequestService.getFunCallTimeout());
            return miniMaxRequestService;
        }
    }
}
