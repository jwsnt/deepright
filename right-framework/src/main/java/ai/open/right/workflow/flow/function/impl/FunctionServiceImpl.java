package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.Function;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
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
import org.springframework.util.Assert;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class FunctionServiceImpl implements FunctionService {

    private static final FunctionConfig EMPTY = new FunctionConfig();

    protected Map<String, Function> functions;

    public Object call(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
        String name = functionConfig != null ? functionConfig.getName(workTask.getWorkflow()) : workTask.getWorkflow();
        if (log.isDebugEnabled()) {
            log.debug("Function name={}", name);
        }
        Function function = this.functions.get(name);
        Assert.notNull(function, "The function can not be empty: " + name);
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig != null ? functionConfig : FunctionServiceImpl.EMPTY)
                .workTask(workTask)
                .build();
        Object result = function.call(functionContext);
        return result != null ? result : "";
    }

    @ConditionalOnProperty(name = "function.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Map<String, Function> functions;

        @Bean
        @ConditionalOnMissingBean(value = FunctionService.class)
        public FunctionService functionService() throws Exception {
            FunctionServiceImpl functionService = new FunctionServiceImpl();
            BeanUtils.copyProperties(this, functionService);
            log.info("FunctionServiceImpl inited={}", functionService.getFunctions().keySet());
            return functionService;
        }
    }
}
