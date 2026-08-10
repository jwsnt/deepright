/// /////////////////////////////////////////////////////////////
/// <dependency>
///     <groupId>org.graalvm.python</groupId>
///     <artifactId>python-embedding</artifactId>
///     <version>24.2.2</version>
///     <scope>compile</scope>
/// </dependency>
/// <dependency>
///     <groupId>org.graalvm.polyglot</groupId>
///     <artifactId>polyglot</artifactId>
///     <version>24.2.2</version>
///     <scope>compile</scope>
/// </dependency>
/// <dependency>
///     <groupId>org.graalvm.polyglot</groupId>
///     <artifactId>python</artifactId>
///     <version>24.2.2</version>
///     <type>pom</type>
/// </dependency>
/// /////////////////////////////////////////////////////////////
package ai.open.right.workflow.flow.script.impl;

import ai.open.right.WorkflowException;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringEscapeUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.pool2.BasePooledObjectFactory;
import org.apache.commons.pool2.PooledObject;
import org.apache.commons.pool2.impl.DefaultPooledObject;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.apache.commons.pool2.impl.GenericObjectPoolConfig;
import org.graalvm.polyglot.Context;
import org.graalvm.python.embedding.GraalPyResources;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.io.Closeable;
import java.io.IOException;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Setter
@Getter
@Slf4j
public class PolyglotService extends AbstractScriptService {

    public static final String PATTERN = "```python\\s*([\\s\\S]*?)```";

    public static final String PREFIX = "```python";

    public static final String SUFFIX = "```";

    public static final String LANG = "python";

    protected GenericObjectPool<PolyglotContext> objectPool;

    protected ExecutorService executorService;

    protected Boolean embedding = true;

    // Polyglot连接池大小
    protected Integer maxTotal = 1;

    // Polyglot连接池最大IDEL数量
    protected Integer maxIdle = 1;

    // Polyglot连接池最小IDEL数量
    protected Integer minIdle = 1;

    // Polyglot获取连接等待时间
    protected Integer borrow = 1000;

    @PostConstruct
    public void init() {
        GenericObjectPoolConfig<PolyglotContext> poolConfig = new GenericObjectPoolConfig<PolyglotContext>();
        poolConfig.setMaxTotal(this.maxTotal);
        poolConfig.setMaxIdle(this.maxIdle);
        poolConfig.setMinIdle(this.minIdle);
        this.objectPool = new GenericObjectPool<PolyglotContext>(new PolyglotContextFactory(this.embedding), poolConfig);
    }

    @PreDestroy
    public void destroy() {
        if (this.objectPool != null) {
            this.objectPool.close();
        }
    }

    @Override
    public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
        String pyScript = this.buildScript(scriptEnv, script);
        if (log.isDebugEnabled()) {
            log.debug("PolyglotService script={}", pyScript);
        }
        Assert.hasText(pyScript, "PolyglotService script can not be empty: " + script);
        Future<String> future = null;
        try {
            return StringUtils.defaultIfEmpty((future = this.executorService.submit(new PyFuture(this.objectPool, this.borrow, pyScript))).get(timeout, TimeUnit.MILLISECONDS), "");
        } catch (Exception e) {
            future.cancel(true);
            throw e;
        }
    }

    protected String buildScript(ScriptEnv scriptEnv, String script) {
        if (CollectionUtils.isEmpty(scriptEnv)) {
            return script;
        }
        // 构建ReScript，加载Env
        StringBuffer envScript = new StringBuffer();
        envScript.append("import os").append(System.lineSeparator());
        for (String key : scriptEnv.keySet()) {
            // os.environ["MY_CUSTOM_ENV"]
            envScript.append("os.environ[\"").append(key).append("\"] = \"").append(StringEscapeUtils.escapeJson(scriptEnv.get(key))).append("\"").append(System.lineSeparator());
        }
        envScript.append(StringUtils.defaultIfBlank(this.extract(script), this.clean(script)));
        return envScript.toString();
    }

    // 提取Python
    protected String extract(String script) {
        Matcher matcher = Pattern.compile(JythonService.PATTERN).matcher(script);
        if (matcher.find()) {
            return matcher.group(1).trim();
        }
        return null;
    }

    protected String clean(String script) {
        script = script.trim();
        if (script.startsWith(PolyglotService.PREFIX) && script.endsWith(PolyglotService.SUFFIX)) {
            script = new StringBuffer(script).delete(script.length() - PolyglotService.SUFFIX.length(), script.length()).delete(0, PolyglotService.PREFIX.length()).toString();
        }
        if (log.isDebugEnabled()) {
            log.debug("PolyglotService script after cleaning={}", script);
        }
        Assert.hasText(script, "PolyglotService script can not be empty: " + script);
        return script;
    }

    public static class PyFuture implements Callable<String> {

        protected final GenericObjectPool<PolyglotContext> objectPool;

        protected final Integer borrow;

        protected final String script;

        public PyFuture(GenericObjectPool<PolyglotContext> objectPool, Integer borrow, String script) {
            this.objectPool = objectPool;
            this.borrow = borrow;
            this.script = script;
        }

        @Override
        public String call() throws Exception {
            PolyglotContext polyglotContext = null;
            try {
                // 清除状态（流和资源限制），但是不清楚全局变量（需要由使用者自行控制全局变量的使用）
                polyglotContext = this.objectPool.borrowObject(this.borrow).reset();
                polyglotContext.getContext().eval(PolyglotService.LANG, this.script);
                String result = polyglotContext.content();
                this.objectPool.returnObject(polyglotContext);
                return StringUtils.defaultIfEmpty(result, "");
            } catch (Exception e) {
                this.release(e, polyglotContext);
                throw e;
            }
        }

        protected void release(Exception e, PolyglotContext polyglotContext) {
            if (polyglotContext != null) {
                try {
                    this.objectPool.invalidateObject(polyglotContext);
                } catch (Exception destroy) {
                    WorkflowException.dolog(e);
                }
            }
        }
    }

    public static class PolyglotContextFactory extends BasePooledObjectFactory<PolyglotContext> {

        protected Boolean embedding = true;

        public PolyglotContextFactory(Boolean embedding) {
            this.embedding = embedding;
        }

        @Override
        public PolyglotContext create() {
            return new PolyglotContext(this.embedding);
        }

        @Override
        public PooledObject<PolyglotContext> wrap(PolyglotContext polyglotContext) {
            return new DefaultPooledObject<PolyglotContext>(polyglotContext);
        }

        @Override
        public void destroyObject(PooledObject<PolyglotContext> p) {
            IOUtils.closeQuietly(p.getObject());
        }
    }

    @Setter
    @Getter
    public static class StringBufferOutputStream extends OutputStream {

        private final StringBuffer buffer = new StringBuffer();

        public StringBufferOutputStream reset() {
            this.buffer.delete(0, this.buffer.length());
            this.buffer.setLength(0);
            return this;
        }

        @Override
        public void write(int b) throws IOException {
            this.buffer.append((char) b);
        }

        @Override
        public void write(byte[] b) throws IOException {
            this.buffer.append(new String(b, StandardCharsets.UTF_8));
        }

        @Override
        public void write(byte[] b, int off, int len) throws IOException {
            this.buffer.append(new String(b, off, len, StandardCharsets.UTF_8));
        }
    }

    @Setter
    @Getter
    public static class PolyglotContext implements Closeable {

        protected StringBufferOutputStream stream;

        protected Context context;

        public PolyglotContext(Boolean embedding) {
            this.stream = new StringBufferOutputStream();
            Context.Builder builder = embedding ? GraalPyResources.contextBuilder() : Context.newBuilder();
            this.context = builder.out(this.stream).build();
            this.context.initialize(PolyglotService.LANG);
        }

        public PolyglotContext eval(String script) throws IOException {
            this.context.eval(PolyglotService.LANG, script);
            return this;
        }

        public PolyglotContext reset() throws Exception {
            this.context.resetLimits();
            this.stream.reset();
            return this;
        }

        public String content() throws Exception {
            return this.stream.getBuffer().toString();
        }

        @Override
        public void close() throws IOException {
            IOUtils.closeQuietly(this.stream);
            if (this.context != null) {
                this.context.close();
            }
        }
    }

    @ConditionalOnProperty(name = "polyglot.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ScriptInitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Value("${polyglot.embedding:true}")
        protected Boolean embedding = true;

        @Value("${polyglot.max.total:20}")
        // Polyglot连接池大小
        protected Integer maxTotal = 1;

        @Value("${polyglot.max.idle:20}")
        // Polyglot连接池最大IDEL数量
        protected Integer maxIdle = 1;

        @Value("${polyglot.min.idle:10}")
        // Polyglot连接池最小IDEL数量
        protected Integer minIdle = 1;

        @Value("${polyglot.borrow:1000}")
        // Polyglot获取连接等待时间
        protected Integer borrow = 1000;

        @Bean
        @ConditionalOnMissingBean(value = PolyglotService.class)
        public PolyglotService polyglotService() throws Exception {
            PolyglotService polyglotService = new PolyglotService();
            BeanUtils.copyProperties(this, polyglotService);
            log.info("PolyglotService inited: maxTotal={},maxIdle={},minIdle={},borrow={}", polyglotService.getMaxTotal(), polyglotService.getMaxIdle(), polyglotService.getMinIdle(), polyglotService.getBorrow());
            return polyglotService;
        }
    }
}