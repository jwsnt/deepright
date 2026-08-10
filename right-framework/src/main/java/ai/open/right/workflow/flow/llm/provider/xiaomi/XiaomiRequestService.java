package ai.open.right.workflow.flow.llm.provider.xiaomi;

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
public class XiaomiRequestService extends OpenAiRequestService implements ProviderRequestModel {

    public static final String NAME = "XiaomiRequestService";

    protected String model;

    // Xiaomi Token
    protected String token;

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

    @Override
    protected void reasoning(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        // https://mimo.mi.com/docs/zh-CN/quick-start/usage-guide/text-generation/deep-thinking
        // completion = client.chat.completions.create(
        //    model="mimo-v2.5-pro",
        //    messages=[
        //        {
        //            "role": "system",
        //            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        //        },
        //        {
        //            "role": "user",
        //            "content": "Introduce machine learning in three sentences."
        //        }
        //    ],
        //    max_completion_tokens=1024,
        //    extra_body={
        //        "thinking": {"type": "enabled"}
        //    }
        //)
        // 兼容标准协议thinking={"type": "enabled"}
        Map<String, Object> thinking = MapUtils.getMap(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING);
        thinking = !MapUtils.isEmpty(thinking) ? thinking : MapUtils.getMap(llmConfig.getAdditional(), ProviderRequestService.KEY_THINKING);
        if (!MapUtils.isEmpty(thinking)) {
            // mimo-v2.5-pro、mimo-v2.5不写thinking不是关闭，而是默认开启
            request.setExtra(ProviderRequestService.KEY_THINKING, thinking);
        }
    }

    @Override
    protected void extra(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
    }

    @ConditionalOnProperty(name = "xiaomi.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Value("${xiaomi.model:mimo-v2.5}")
        protected String model;

        @Value("${xiaomi.token:}")
        // Xiaomi Token
        protected String token;

        @Bean(name = XiaomiRequestService.NAME)
        @ConditionalOnMissingBean(name = XiaomiRequestService.NAME)
        public XiaomiRequestService xiaomiRequestService() throws Exception {
            XiaomiRequestService xiaomiRequestService = new XiaomiRequestService();
            BeanUtils.copyProperties(this, xiaomiRequestService);
            log.info("XiaomiRequestService inited. model={}, token={}, timeout={}", xiaomiRequestService.getModel(), StringUtils.repeat("*", StringUtils.length(xiaomiRequestService.getToken())), xiaomiRequestService.getFunCallTimeout());
            return xiaomiRequestService;
        }
    }
}
