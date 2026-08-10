package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.JavaScriptService;
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
// 自定义JS AsyncFunction
public class JavaScriptFunction extends ScriptFunction {

    public static final String NAME = "function.javascript";

    protected JavaScriptService javaScriptService;

    // 调用JS超时
    protected Integer timeout;

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        // 需要加载的资源
        Assert.isTrue(functionContext.getFunctionConfig().hasResource(), "JavaScript's resource can not be empty, please check config");
        String key = functionContext.getFunctionConfig().getResource();
        return this.javaScriptService.run(this.buildEnv(functionContext.getFunctionConfig(), functionContext.getWorkTask()), this.getScript(key), functionContext.getFunctionConfig().getTimeout(this.timeout));
    }

    @ConditionalOnProperty(name = "javascript.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected JavaScriptService javaScriptService;

        @Value("${function.javascript.timeout:1800000}")
        // 调用JS超时
        protected Integer timeout;

        @Bean(JavaScriptFunction.NAME)
        @ConditionalOnMissingBean(name = JavaScriptFunction.NAME)
        public JavaScriptFunction javaScriptFunction() throws Exception {
            JavaScriptFunction javaScriptFunction = new JavaScriptFunction();
            BeanUtils.copyProperties(this, javaScriptFunction);
            log.info("JavaScriptFunction inited: timeout={}", javaScriptFunction.getTimeout());
            return javaScriptFunction;
        }
    }
}
