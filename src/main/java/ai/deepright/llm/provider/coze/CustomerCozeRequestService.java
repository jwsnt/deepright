package ai.deepright.llm.provider.coze;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.coze.CozeRequest;
import ai.open.right.workflow.flow.llm.provider.coze.CozeRequestService;
import ai.deepright.llm.RetryUtils;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class CustomerCozeRequestService extends CozeRequestService {

    @Override
    protected void storeHistoryQuery(CozeRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        RetryUtils.storeQuery(request, llmConfig, this.historyStore, this.buildHistoryQuery(request, llmConfig));
    }

    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Override
        @Bean(name = CozeRequestService.NAME)
        @ConditionalOnMissingBean(name = CozeRequestService.NAME)
        public CozeRequestService cozeRequestService() throws Exception {
            CustomerCozeRequestService cozeRequestService = new CustomerCozeRequestService();
            BeanUtils.copyProperties(this, cozeRequestService);
            log.info("CustomerCozeRequestService inited, timeout={}", cozeRequestService.getFunCallTimeout());
            return cozeRequestService;
        }
    }
}
