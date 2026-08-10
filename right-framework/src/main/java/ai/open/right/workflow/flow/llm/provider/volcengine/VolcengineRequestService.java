package ai.open.right.workflow.flow.llm.provider.volcengine;

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
public class VolcengineRequestService extends OpenAiRequestService implements ProviderRequestModel {

    public static final String NAME = "VolcengineRequestService";

    protected String reasoningEffort;

    // 火山引擎模型
    protected String model = "doubao-seed-2-1-turbo-260628";

    // 火山引擎Token
    protected String token;

    @Override
    protected void request(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setFunCallStream(false);
    }

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
    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://docs.volcengine.com/docs/82379/1449737?lang=zh
        // completion = client.chat.completions.create(
        //    # Replace with Model ID
        //    model = "doubao-seed-2-1-pro-260628",
        //    messages=[
        //        {"role": "user", "content": "我要研究深度思考模型与非深度思考模型区别的课题，体现出我的专业性"}
        //    ],
        //     thinking={
        //         "type": "disabled", # 不使用深度思考能力
        //         # "type": "enabled", # 使用深度思考能力
        //         # "type": "auto", # 模型自行判断是否使用深度思考能力
        //     },
        //)
        //
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        if (!MapUtils.isEmpty(thinking)) {
            // 不能将不写enable_thinking视作统一关闭
            request.setExtra(ProviderRequestService.KEY_THINKING, thinking);
            if (StringUtils.equalsAnyIgnoreCase(MapUtils.getString(thinking, "type"), "enabled", "adaptive")) {
                String reasoningEffort = MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT);
                reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : MapUtils.getString(llmConfig.getAdditional(), ProviderRequestService.KEY_REASONING_EFFORT);
                reasoningEffort = !StringUtils.isEmpty(reasoningEffort) ? reasoningEffort : this.reasoningEffort;
                request.setReasoningEffort(reasoningEffort);
            }
        }
    }

    @Override
    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    @ConditionalOnProperty(name = "volcengine.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${volcengine.model.reasoningEffort:low}")
        protected String reasoningEffort;

        @Value("${volcengine.model:doubao-seed-2-1-turbo-260628}")
        // 火山引擎模型
        protected String model = "doubao-seed-2-1-turbo-260628";

        @Value("${volcengine.token:}")
        // 火山引擎Token
        protected String token;

        @Bean(name = VolcengineRequestService.NAME)
        @ConditionalOnMissingBean(name = VolcengineRequestService.NAME)
        public VolcengineRequestService volcengineRequestService() throws Exception {
            VolcengineRequestService volcengineRequestService = new VolcengineRequestService();
            BeanUtils.copyProperties(this, volcengineRequestService);
            log.info("VolcengineRequestService inited, model={}, token={}, timeout={}", volcengineRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(volcengineRequestService.getToken())), volcengineRequestService.getFunCallTimeout());
            return volcengineRequestService;
        }
    }
}