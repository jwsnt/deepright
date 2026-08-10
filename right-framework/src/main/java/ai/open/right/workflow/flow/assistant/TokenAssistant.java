package ai.open.right.workflow.flow.assistant;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Setter
@Getter
public class TokenAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-token";

    protected TokenStatistic tokenStatistic;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildTokenQuery(workflowConfig, workTask, this.buildTokenData(workflowConfig, workTask)));
    }

    protected String buildTokenQuery(WorkflowConfig workflowConfig, WorkflowTask workTask, Object token) throws Exception {
        return JsonUtils.write(token);
    }

    protected Object buildTokenData(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        return this.tokenStatistic.read(workTask);
    }

    @ConditionalOnProperty(name = "token.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected TokenStatistic tokenStatistic;

        @Bean(TokenAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = TokenAssistant.WORKFLOW_NAME)
        public TokenAssistant tokenAssistant() throws Exception {
            TokenAssistant tokenAssistant = new TokenAssistant();
            BeanUtils.copyProperties(this, tokenAssistant);
            log.info("TokenAssistant inited");
            return tokenAssistant;
        }
    }
}
