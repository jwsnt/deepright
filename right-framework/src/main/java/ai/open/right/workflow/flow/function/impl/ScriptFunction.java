package ai.open.right.workflow.flow.function.impl;

import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.Function;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.script.impl.ScriptEnv;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.factory.annotation.Autowired;

import java.io.BufferedInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutionException;

@Setter
@Getter
@Slf4j
abstract public class ScriptFunction implements Function {

    // MaximumSize由使用者控制
    protected final Cache<String, String> scriptCache = CacheBuilder.newBuilder().maximumSize(Integer.MAX_VALUE).build();

    protected ResourceService resourceService;

    // 构建脚本环境变量
    protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
        ScriptEnv scriptEnv = new ScriptEnv(workTask);
        if (functionConfig.hasEnvironment()) {
            scriptEnv.env(functionConfig.getEnvironment());
        }
        if (log.isDebugEnabled()) {
            log.debug("Script env={}", scriptEnv);
        }
        return scriptEnv;
    }

    protected String getScript(String key) throws ExecutionException {
        if (log.isInfoEnabled()) {
            log.info("Get script key={}", key);
        }
        return this.scriptCache.get(key, new ScriptCallable(this.resourceService, key));
    }

    public static class ScriptCallable implements Callable<String> {

        protected final ResourceService resourceService;

        protected final String key;

        public ScriptCallable(ResourceService resourceService, String key) {
            this.resourceService = resourceService;
            this.key = key;
        }

        @Override
        public String call() throws Exception {
            try (InputStream input = new BufferedInputStream(this.resourceService.url(this.key).openStream())) {
                return IOUtils.toString(input, StandardCharsets.UTF_8);
            }
        }
    }

    @Getter
    @Setter
    public static class DefInitConfig {

        @Autowired
        protected ResourceService resourceService;
    }
}
