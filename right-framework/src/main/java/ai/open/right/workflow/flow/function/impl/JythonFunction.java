package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.JythonService;
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
public class JythonFunction extends ScriptFunction {

    public static final String NAME = "function.jython";

    protected JythonService jythonService;

    // 调用Jython超时
    protected Integer timeout;

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        // 需要加载的资源
        Assert.isTrue(functionContext.getFunctionConfig().hasResource(), "Jython's resource can not be empty, please check config");
        String key = functionContext.getFunctionConfig().getResource();
        return this.jythonService.run(this.buildEnv(functionContext.getFunctionConfig(), functionContext.getWorkTask()), this.getScript(key), functionContext.getFunctionConfig().getTimeout(this.timeout));
    }

    @ConditionalOnProperty(name = "jython.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected JythonService jythonService;

        @Value("${function.jython.timeout:1800000}")
        // 调用Jython超时
        protected Integer timeout;

        @Bean(JythonFunction.NAME)
        @ConditionalOnMissingBean(name = JythonFunction.NAME)
        public JythonFunction jythonFunction() throws Exception {
            JythonFunction jythonFunction = new JythonFunction();
            BeanUtils.copyProperties(this, jythonFunction);
            log.info("JythonFunction inited: timeout={}", jythonFunction.getTimeout());
            return jythonFunction;
        }
    }
}
