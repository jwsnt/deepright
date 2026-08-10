package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.PythonService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Setter
@Getter
@Slf4j
public class PythonFunction extends ScriptFunction {

    public static final String NAME = "function.python";

    protected PythonService pythonService;

    // 调用Python超时
    protected Integer timeout;

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        // 需要加载的资源
        Assert.isTrue(functionContext.getFunctionConfig().hasResource(), "Python's resource can not be empty, please check config");
        String key = functionContext.getFunctionConfig().getResource();
        return this.pythonService.run(this.buildEnv(functionContext.getFunctionConfig(), functionContext.getWorkTask()), this.getScript(key), functionContext.getFunctionConfig().getTimeout(this.timeout));
    }

    @ConditionalOnProperty(name = "python.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected PythonService pythonService;

        @Value("${function.python.timeout:1800000}")
        // 调用Python超时
        protected Integer timeout;

        @Bean(PythonFunction.NAME)
        @ConditionalOnMissingBean(name = PythonFunction.NAME)
        public PythonFunction pythonFunction() throws Exception {
            PythonFunction pythonFunction = new PythonFunction();
            BeanUtils.copyProperties(this, pythonFunction);
            log.info("PythonFunction inited: timeout={}", pythonFunction.getTimeout());
            return pythonFunction;
        }
    }
}
