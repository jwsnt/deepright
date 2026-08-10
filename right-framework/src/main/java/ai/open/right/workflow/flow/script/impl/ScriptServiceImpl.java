package ai.open.right.workflow.flow.script.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.script.ScriptConfig;
import ai.open.right.workflow.flow.script.ScriptService;
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

@Slf4j
@Setter
@Getter
public class ScriptServiceImpl implements ScriptService {

    public static final String NAME = "ScriptServiceImpl";

    protected JavaScriptService javaScriptService;

    protected PolyglotService polyglotService;

    protected CommandService commandService;

    protected JythonService jythonService;

    protected PythonService pythonService;

    protected LuaService luaService;

    @Override
    // 由使用者保证沙箱安全
    public String run(ScriptConfig scriptConfig, WorkflowTask workTask) throws Exception {
        switch (scriptConfig != null ? scriptConfig.getEngine() : ScriptConfig.ENGINE_PYTHON) {
            case ScriptConfig.ENGINE_JAVASCRIPT:
                Assert.notNull(this.javaScriptService, "The java script service can not be empty");
                return this.javaScriptService.run(scriptConfig, workTask);
            case ScriptConfig.ENGINE_POLYGLOT:
                Assert.notNull(this.polyglotService, "The polyglot service can not be empty");
                return this.polyglotService.run(scriptConfig, workTask);
            case ScriptConfig.ENGINE_COMMAND:
                Assert.notNull(this.commandService, "The command service can not be empty");
                return this.commandService.run(scriptConfig, workTask);
            case ScriptConfig.ENGINE_PYTHON:
                Assert.notNull(this.pythonService, "The python service can not be empty");
                return this.pythonService.run(scriptConfig, workTask);
            case ScriptConfig.ENGINE_JYTHON:
                Assert.notNull(this.jythonService, "The jython service can not be empty");
                return this.jythonService.run(scriptConfig, workTask);
            case ScriptConfig.ENGINE_LUA:
                Assert.notNull(this.luaService, "The lua service can not be empty");
                return this.luaService.run(scriptConfig, workTask);
            default:
                if (log.isDebugEnabled()) {
                    log.debug("Script Service will use default engine python");
                }
                // 默认为Python
                Assert.notNull(this.pythonService, "The python service can not be empty");
                return this.pythonService.run(scriptConfig, workTask);
        }
    }

    @ConditionalOnProperty(name = "script.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected JavaScriptService javaScriptService;

        @Autowired(required = false)
        protected PolyglotService polyglotService;

        @Autowired(required = false)
        protected CommandService commandService;

        @Autowired(required = false)
        protected JythonService jythonService;

        @Autowired(required = false)
        protected PythonService pythonService;

        @Autowired(required = false)
        protected LuaService luaService;

        @Bean(name = ScriptServiceImpl.NAME)
        @ConditionalOnMissingBean(name = ScriptServiceImpl.NAME)
        public ScriptService scriptService() throws Exception {
            ScriptServiceImpl scriptService = new ScriptServiceImpl();
            BeanUtils.copyProperties(this, scriptService);
            log.info("ScriptServiceImpl inited");
            return scriptService;
        }
    }
}
