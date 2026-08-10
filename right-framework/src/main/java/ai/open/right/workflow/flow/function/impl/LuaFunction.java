package ai.open.right.workflow.flow.function.impl;

import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.LuaService;
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
public class LuaFunction extends ScriptFunction {

    public static final String NAME = "function.lua";

    protected LuaService luaService;

    // 调用Lua超时
    protected Integer timeout;

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        // 需要加载的资源
        Assert.isTrue(functionContext.getFunctionConfig().hasResource(), "Lua' resource can not be empty, please check config");
        String key = functionContext.getFunctionConfig().getResource();
        return this.luaService.run(this.buildEnv(functionContext.getFunctionConfig(), functionContext.getWorkTask()), this.getScript(key), functionContext.getFunctionConfig().getTimeout(this.timeout));
    }

    @ConditionalOnProperty(name = "lua.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected LuaService luaService;

        @Value("${function.lua.timeout:1800000}")
        // 调用Lua超时
        protected Integer timeout;

        @Bean(LuaFunction.NAME)
        @ConditionalOnMissingBean(name = LuaFunction.NAME)
        public LuaFunction luaFunction() throws Exception {
            LuaFunction luaFunction = new LuaFunction();
            BeanUtils.copyProperties(this, luaFunction);
            log.info("LuaFunction inited: timeout={}", luaFunction.getTimeout());
            return luaFunction;
        }
    }
}
