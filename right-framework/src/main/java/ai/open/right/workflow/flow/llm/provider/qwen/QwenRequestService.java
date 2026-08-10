package ai.open.right.workflow.flow.llm.provider.qwen;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequestService;
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
public class QwenRequestService extends OpenAiRequestService implements ProviderRequestModel {

    public static final String NAME = "QwenRequestService";

    // Qwen模型
    protected String model = "qwen3.7-plus";

    // Qwen Token
    protected String token;

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return this.model;
    }

    @Override
    protected void request(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setOpenAiMedia(OpenAiRequest.DefaultMedia.DEFAULT);
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
    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://modelstudio.console.alibabacloud.com/ap-southeast-1/?tab=api#/api/?type=model&url=3016807
        // completion = client.chat.completions.create(
        // # This example uses qwen-plus. You can replace it with another model name as needed. Model list: https://www.alibabacloud.com/help/en/model-studio/getting-started/models
        // model="qwen-plus",
        // messages=[
        //  {"role": "system", "content": "You are a helpful assistant."},
        //  {"role": "user", "content": "Who are you?"},
        // ],
        // # extra_body={"enable_thinking": False},
        // # extra_body={"reasoning_effort": "high"}
        //)
        // 兼容标准协议thinking={"type": "enabled"}
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        if (!MapUtils.isEmpty(thinking)) {
            if (StringUtils.equalsAnyIgnoreCase(MapUtils.getString(thinking, "type"), "enabled", "adaptive")) {
                String reasoningEffort = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
                reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
                reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : this.reasoningEffort;
                request.setReasoningEffort(reasoningEffort);
                request.setExtra("enable_thinking", true);
            } else {
                request.setExtra("enable_thinking", false);
            }
        }
    }

    @Override
    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    @ConditionalOnProperty(name = "qwen.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${qwen.model:qwen3.7-plus}")
        // Qwen模型
        protected String model = "qwen3.7-plus";

        @Value("${qwen.token:}")
        // Qwen Token
        protected String token;

        @Bean(name = QwenRequestService.NAME)
        @ConditionalOnMissingBean(name = QwenRequestService.NAME)
        public QwenRequestService qwenRequestService() throws Exception {
            QwenRequestService qwenRequestService = new QwenRequestService();
            BeanUtils.copyProperties(this, qwenRequestService);
            log.info("QwenRequestService inited, timeout={}", qwenRequestService.getFunCallTimeout());
            return qwenRequestService;
        }
    }
}