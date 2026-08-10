package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.PolyglotService;
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
public class PolyglotFunction extends ScriptFunction {

    public static final String NAME = "function.polyglot";

    protected PolyglotService polyglotService;

    // 调用Polyglot超时
    protected Integer timeout;

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        // 需要加载的资源
        Assert.isTrue(functionContext.getFunctionConfig().hasResource(), "Polyglot's resource can not be empty, please check config");
        String key = functionContext.getFunctionConfig().getResource();
        return this.polyglotService.run(this.buildEnv(functionContext.getFunctionConfig(), functionContext.getWorkTask()), this.getScript(key), functionContext.getFunctionConfig().getTimeout(this.timeout));
    }

    @ConditionalOnProperty(name = "polyglot.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected PolyglotService polyglotService;

        @Value("${function.polyglot.timeout:1800000}")
        // 调用Polyglot超时
        protected Integer timeout;

        @Bean(PolyglotFunction.NAME)
        @ConditionalOnMissingBean(name = PolyglotFunction.NAME)
        public PolyglotFunction polyglotFunction() throws Exception {
            PolyglotFunction polyglotFunction = new PolyglotFunction();
            BeanUtils.copyProperties(this, polyglotFunction);
            log.info("PolyglotFunction inited: timeout={}", polyglotFunction.getTimeout());
            return polyglotFunction;
        }
    }
}
