/// /////////////////////////////////////////////////////////////
/// <dependency>
///     <groupId>org.luaj</groupId>
///     <artifactId>luaj-jse</artifactId>
///     <version>3.0.1</version>
///     <scope>compile</scope>
/// </dependency>
/// /////////////////////////////////////////////////////////////
package ai.open.right.workflow.flow.script.impl;

import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.pool2.BasePooledObjectFactory;
import org.apache.commons.pool2.PooledObject;
import org.apache.commons.pool2.impl.DefaultPooledObject;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.apache.commons.pool2.impl.GenericObjectPoolConfig;
import org.luaj.vm2.Globals;
import org.luaj.vm2.LuaError;
import org.luaj.vm2.LuaValue;
import org.luaj.vm2.Varargs;
import org.luaj.vm2.lib.VarArgFunction;
import org.luaj.vm2.lib.jse.JsePlatform;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

@Setter
@Getter
@Slf4j
public class LuaService extends AbstractScriptService {

    public static final String PREFIX = "```lua";

    public static final String SUFFIX = "```";

    protected GenericObjectPool<Globals> objectPool;

    protected ResourceService resourceService;

    protected ExecutorService executorService;

    // Lua连接池大小
    protected Integer maxTotal = 1;

    // Jython连接池最大IDEL数量
    protected Integer maxIdle = 1;

    // Jython连接池最小IDEL数量
    protected Integer minIdle = 1;

    // Jython连接池最小IDEL数量
    protected Integer borrow = 1000;

    protected String dkjson;

    @PostConstruct
    public void init() throws Exception {
        try (InputStream input = this.resourceService.url("classpath:dkjson.lua").openStream()) {
            // 加载外部dkjson包
            this.dkjson = IOUtils.toString(input, StandardCharsets.UTF_8);
            GenericObjectPoolConfig<Globals> poolConfig = new GenericObjectPoolConfig<Globals>();
            poolConfig.setMaxTotal(this.maxTotal);
            poolConfig.setMaxIdle(this.maxIdle);
            poolConfig.setMinIdle(this.minIdle);
            this.objectPool = new GenericObjectPool<Globals>(new LuaJGlobalsFactory(), poolConfig);
        }
    }

    @PreDestroy
    public void destroy() {
        if (this.objectPool != null) {
            this.objectPool.close();
        }
    }

    @Override
    public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
        String luaScript = StringUtils.defaultIfBlank(this.clean(script), script);
        if (log.isInfoEnabled()) {
            log.info("LuaService script={}", luaScript);
        }
        Assert.hasText(luaScript, "LuaService script can not be empty: " + script);
        Future<String> future = null;
        try {
            return StringUtils.defaultIfEmpty((future = this.executorService.submit(new LuaFuture(this.objectPool, scriptEnv, this.borrow, luaScript, this.dkjson))).get(timeout, TimeUnit.MILLISECONDS), "");
        } catch (Exception e) {
            future.cancel(true);
            if (e.getCause() != null) {
                if (LuaError.class.isAssignableFrom(e.getCause().getClass())) {
                    LuaError luaError = LuaError.class.cast(e.getCause());
                    throw WorkflowException.create(luaError.getMessage());
                }
            }
            throw e;
        }
    }

    protected String clean(String script) {
        script = script.trim();
        if (script.startsWith(LuaService.PREFIX) && script.endsWith(LuaService.SUFFIX)) {
            script = new StringBuffer(script)
                    .delete(script.length() - LuaService.SUFFIX.length(), script.length())
                    .delete(0, LuaService.PREFIX.length()).toString();
        }
        if (log.isDebugEnabled()) {
            log.debug("LuaService script after cleaning={}", script);
        }
        Assert.hasText(script, "LuaService script can not be empty");
        return script;
    }

    public static class LuaFuture implements Callable<String> {

        protected final GenericObjectPool<Globals> objectPool;

        protected final ScriptEnv scriptEnv;

        protected final Integer borrow;

        protected final String script;

        protected final String dkjson;

        public LuaFuture(GenericObjectPool<Globals> objectPool, ScriptEnv scriptEnv, Integer borrow, String script, String dkjson) {
            this.objectPool = objectPool;
            this.scriptEnv = scriptEnv;
            this.borrow = borrow;
            this.script = script;
            this.dkjson = dkjson;
        }

        @Override
        public String call() throws Exception {
            Globals globals = null;
            try {
                globals = this.objectPool.borrowObject(this.borrow);
                for (String key : this.scriptEnv.keySet()) {
                    globals.set(key, this.scriptEnv.get(key));
                }
                // dkjson
                globals.load(this.dkjson).call();
                try (PrintFunction print = new PrintFunction()) {
                    globals.set("print", print);
                    globals.load(this.script).call();
                    this.objectPool.returnObject(globals);
                    return print.buildContent();
                }
            } catch (Exception e) {
                this.release(e, globals);
                throw e;
            }
        }

        protected void release(Exception e, Globals globals) {
            if (globals != null) {
                try {
                    this.objectPool.invalidateObject(globals);
                } catch (Exception destroy) {
                    WorkflowException.dolog(e);
                }
            }
        }
    }

    public static class PrintFunction extends VarArgFunction implements Closeable {

        private final ByteArrayOutputStream output = new ByteArrayOutputStream();

        private final PrintStream print = new PrintStream(new BufferedOutputStream(this.output));

        @Override
        public LuaValue invoke(Varargs args) {
            for (int i = 1; i <= args.narg(); i++) {
                this.print.print(args.arg(i).tojstring());
                if (i < args.narg()) {
                    this.print.print("\t");
                }
            }
            this.print.println();
            return LuaValue.NIL;
        }

        public String buildContent() {
            this.print.flush();
            return this.output.toString(StandardCharsets.UTF_8);
        }

        @Override
        public void close() throws IOException {
            IOUtils.closeQuietly(this.output);
            IOUtils.closeQuietly(this.print);
        }
    }

    public static class LuaJGlobalsFactory extends BasePooledObjectFactory<Globals> {

        @Override
        public Globals create() {
            return JsePlatform.standardGlobals();
        }

        @Override
        public PooledObject<Globals> wrap(Globals globals) {
            return new DefaultPooledObject<Globals>(globals);
        }

        @Override
        public void destroyObject(PooledObject<Globals> p) {
        }
    }

    @ConditionalOnProperty(name = "lua.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ScriptInitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected ResourceService resourceService;

        @Value("${lua.max.total:20}")
        // Lua连接池大小
        protected Integer maxTotal = 1;

        @Value("${lua.max.idle:20}")
        // Jython连接池最大IDEL数量
        protected Integer maxIdle = 1;

        @Value("${lua.min.idle:10}")
        // Jython连接池最小IDEL数量
        protected Integer minIdle = 1;

        @Value("${lua.borrow:1000}")
        // Jython连接池最小IDEL数量
        protected Integer borrow = 1000;

        @Bean
        @ConditionalOnMissingBean(value = LuaService.class)
        public LuaService luaService() throws Exception {
            LuaService luaService = new LuaService();
            BeanUtils.copyProperties(this, luaService);
            log.info("LuaService inited: maxTotal={},maxIdle={},minIdle={},borrow={}", luaService.getMaxTotal(), luaService.getMaxIdle(), luaService.getMinIdle(), luaService.getBorrow());
            return luaService;
        }
    }
}
