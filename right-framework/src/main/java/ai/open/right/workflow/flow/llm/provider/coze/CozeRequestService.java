package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Getter
@Setter
public class CozeRequestService extends ProviderRequestService<CozeRequest> implements ProviderRequestModel {

    public static final String NAME = "CozeRequestService";

    public static final String KEY_BOT = "botId";

    public static final String MODEL = "coze";

    @Override
    protected CozeRequest build() throws Exception {
        return new CozeRequest();
    }

    @Override
    public CozeRequest config(LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        CozeRequest request = super.config(llmConfig, llmQuery);
        CozeRequestChecker.check(request);
        return request;
    }

    @Override
    protected void request(CozeRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        super.request(request, llmConfig, llmQuery);
        request.setBotId(MapUtils.getString(llmQuery.getMetadata(), ProviderRequestService.KEY_INTERNAL + CozeRequestService.KEY_BOT, String.valueOf(llmConfig.getAdditional().get(CozeRequestService.KEY_BOT))));
        request.setApi(ProviderRequest.REQUEST_COZE);
    }

    @Override
    protected String defModel(WorkflowTask workTask) throws Exception {
        return  StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), "__model"), CozeRequestService.MODEL);
    }

    @Override
    public String getModel(WorkflowTask workTask) throws Exception {
        return CozeRequestService.MODEL;
    }

    public static class CozeRequestChecker {

        public static void check(CozeRequest cozeRequest) {
            Assert.hasText(cozeRequest.getToken(), "Token can not be empty");
            Assert.hasText(cozeRequest.getBotId(), "Bot id can not be empty");
        }
    }

    @ConditionalOnProperty(name = "coze.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ProviderRequestInitConfig {

        @Bean(name = CozeRequestService.NAME)
        @ConditionalOnMissingBean(name = CozeRequestService.NAME)
        public CozeRequestService cozeRequestService() throws Exception {
            CozeRequestService cozeRequestService = new CozeRequestService();
            BeanUtils.copyProperties(this, cozeRequestService);
            log.info("CozeRequestService inited. timeout={}", cozeRequestService.getFunCallTimeout());
            return cozeRequestService;
        }
    }
}
