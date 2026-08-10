/// /////////////////////////////////////////////////////////////
/// <dependency>
///     <groupId>org.python</groupId>
///     <artifactId>jython-standalone</artifactId>
///     <version>2.7.4</version>
///     <scope>compile</scope>
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
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.pool2.BasePooledObjectFactory;
import org.apache.commons.pool2.PooledObject;
import org.apache.commons.pool2.impl.DefaultPooledObject;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.apache.commons.pool2.impl.GenericObjectPoolConfig;
import org.python.core.Py;
import org.python.util.PythonInterpreter;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedWriter;
import java.io.ByteArrayOutputStream;
import java.io.OutputStreamWriter;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Setter
@Getter
@Slf4j
public class JythonService extends AbstractScriptService {

    public static final String COMPATIBILITY_1 = "# -*- coding: utf-8 -*-";

    public static final String COMPATIBILITY_2 = "from __future__ import print_function";

    public static final String PATTERN = "```python\\s*([\\s\\S]*?)```";

    public static final String PREFIX = "```python";

    public static final String SUFFIX = "```";

    protected GenericObjectPool<PythonInterpreter> objectPool;

    protected ExecutorService executorService;

    // Jython连接池大小
    protected Integer maxTotal = 1;

    // Jython连接池最大IDEL数量
    protected Integer maxIdle = 1;

    // Jython连接池最小IDEL数量
    protected Integer minIdle = 1;

    // Jython获取连接等待时间
    protected Integer borrow = 1000;

    @PostConstruct
    public void init() {
        GenericObjectPoolConfig<PythonInterpreter> poolConfig = new GenericObjectPoolConfig<PythonInterpreter>();
        poolConfig.setMaxTotal(this.maxTotal);
        poolConfig.setMaxIdle(this.maxIdle);
        poolConfig.setMinIdle(this.minIdle);
        this.objectPool = new GenericObjectPool<PythonInterpreter>(new PythonInterpreterFactory(), poolConfig);
    }

    @PreDestroy
    public void destroy() {
        if (this.objectPool != null) {
            this.objectPool.close();
        }
    }

    @Override
    public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
        String jyScript = StringUtils.defaultIfBlank(this.extract(script), this.clean(script));
        if (log.isInfoEnabled()) {
            log.info("JythonService script={}", jyScript);
        }
        Assert.hasText(jyScript, "JythonService script can not be empty: " + script);
        Future<String> future = null;
        try {
            return StringUtils.defaultIfEmpty((future = this.executorService.submit(new JyFuture(this.objectPool, scriptEnv, this.borrow, this.append(jyScript)))).get(timeout, TimeUnit.MILLISECONDS), "");
        } catch (TimeoutException e) {
            future.cancel(true);
            throw e;
        }
    }

    // 提取Python
    protected String extract(String script) {
        Matcher matcher = Pattern.compile(JythonService.PATTERN).matcher(script);
        if (matcher.find()) {
            return matcher.group(1).trim();
        }
        return null;
    }

    protected String append(String script) {
        StringBuffer buffer = new StringBuffer(JythonService.COMPATIBILITY_1).append(System.lineSeparator());
        buffer.append(JythonService.COMPATIBILITY_2).append(System.lineSeparator()).append(script);
        return buffer.toString();
    }

    protected String clean(String script) {
        script = script.trim();
        if (script.startsWith(JythonService.PREFIX) && script.endsWith(JythonService.SUFFIX)) {
            script = new StringBuffer(script)
                    .delete(script.length() - JythonService.SUFFIX.length(), script.length())
                    .delete(0, JythonService.PREFIX.length()).toString();
        }
        if (log.isDebugEnabled()) {
            log.debug("JythonService script after cleaning={}", script);
        }
        Assert.hasText(script, "JythonService script can not be empty: " + script);
        return script;
    }

    public static class JyFuture implements Callable<String> {

        protected final GenericObjectPool<PythonInterpreter> objectPool;

        protected final ScriptEnv scriptEnv;

        protected final Integer borrow;

        protected final String script;

        public JyFuture(GenericObjectPool<PythonInterpreter> objectPool, ScriptEnv scriptEnv, Integer borrow, String script) {
            this.objectPool = objectPool;
            this.scriptEnv = scriptEnv;
            this.borrow = borrow;
            this.script = script;
        }

        @Override
        public String call() throws Exception {
            PythonInterpreter interpreter = null;
            try {
                interpreter = this.objectPool.borrowObject(this.borrow);
                for (String key : this.scriptEnv.keySet()) {
                    interpreter.set(key, this.scriptEnv.get(key));
                }
                try (ByteArrayOutputStream output = new ByteArrayOutputStream()) {
                    try (BufferedWriter buffer = new BufferedWriter(new OutputStreamWriter(output, StandardCharsets.UTF_8))) {
                        interpreter.setOut(buffer);
                        interpreter.setErr(buffer);
                        interpreter.exec(Py.newStringUTF8(this.script));
                    }
                    interpreter.cleanup();
                    this.objectPool.returnObject(interpreter);
                    return StringUtils.defaultIfEmpty(output.toString(StandardCharsets.UTF_8), "");
                }
            } catch (Exception e) {
                this.release(e, interpreter);
                throw e;
            }
        }

        protected void release(Exception e, PythonInterpreter interpreter) {
            if (interpreter != null) {
                try {
                    this.objectPool.invalidateObject(interpreter);
                } catch (Exception destroy) {
                    WorkflowException.dolog(e);
                }
            }
        }
    }

    public static class PythonInterpreterFactory extends BasePooledObjectFactory<PythonInterpreter> {

        @Override
        public PythonInterpreter create() {
            return new PythonInterpreter();
        }

        @Override
        public PooledObject<PythonInterpreter> wrap(PythonInterpreter interpreter) {
            return new DefaultPooledObject<PythonInterpreter>(interpreter);
        }

        @Override
        public void destroyObject(PooledObject<PythonInterpreter> p) {
            IOUtils.closeQuietly(p.getObject());
        }
    }

    @ConditionalOnProperty(name = "jython.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ScriptInitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Value("${jython.max.total:20}")
        // Jython连接池大小
        protected Integer maxTotal = 1;

        @Value("${jython.max.idle:20}")
        // Jython连接池最大IDEL数量
        protected Integer maxIdle = 1;

        @Value("${jython.min.idle:10}")
        // Jython连接池最小IDEL数量
        protected Integer minIdle = 1;

        @Value("${jython.borrow:1000}")
        // Jython获取连接等待时间
        protected Integer borrow = 1000;

        @Bean
        @ConditionalOnMissingBean(value = JythonService.class)
        public JythonService jythonService() throws Exception {
            JythonService jythonService = new JythonService();
            BeanUtils.copyProperties(this, jythonService);
            log.info("JythonService inited: maxTotal={},maxIdle={},minIdle={},borrow={}", jythonService.getMaxTotal(), jythonService.getMaxIdle(), jythonService.getMinIdle(), jythonService.getBorrow());
            return jythonService;
        }
    }
}