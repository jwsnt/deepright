package ai.open.right.workflow.flow.assistant;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionResponse;
import ai.open.right.workflow.flow.function.FunctionService;
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
// 自定义功能（代码能力）
public class FunctionAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-function";

    protected FunctionService functionService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Object response = this.functionService.call(workflowConfig.getFunctionConfig(), workTask);
        FunctionConfig functionConfig = workflowConfig.getFunctionConfig();
        // 如果配置了Original=True则传递本次思考原始Query作为下次思考链（Workflow）的Query
        if (response != null && response.getClass().isAssignableFrom(FunctionResponse.class)) {
            // 如果返回为FunctionResponse则传递该对象Meta、Code
            FunctionResponse functionResponse = FunctionResponse.class.cast(response);
            if (log.isDebugEnabled()) {
                log.debug("AsyncFunction response={}", functionResponse);
            }
            String query = functionConfig != null ? (functionConfig.getOriginal() ? workTask.getQuery() : JsonUtils.write(functionResponse.getContent())) : JsonUtils.write(functionResponse.getContent());
            this.chainOr2Endpoint(workflowConfig, workTask, functionResponse.getMetadata(), query, functionResponse.getCode());
        } else {
            String query = functionConfig != null ? functionConfig.getOriginal() ? workTask.getQuery() : JsonUtils.write(response) : JsonUtils.write(response);
            this.chainOr2Endpoint(workflowConfig, workTask, query);
        }
    }

    @ConditionalOnProperty(name = "function.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected FunctionService functionService;

        @Bean(FunctionAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = FunctionAssistant.WORKFLOW_NAME)
        public FunctionAssistant functionAssistant() throws Exception {
            FunctionAssistant functionAssistant = new FunctionAssistant();
            BeanUtils.copyProperties(this, functionAssistant);
            log.info("FunctionAssistant inited");
            return functionAssistant;
        }
    }
}
