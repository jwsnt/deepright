////////////////////////////////////////////////////////////////
///<dependency>
///     <groupId>org.mozilla</groupId>
///     <artifactId>rhino</artifactId>
///     <version>1.8.0</version>
///     <scope>compile</scope>
/// </dependency>
////////////////////////////////////////////////////////////////
package ai.open.right.workflow.flow.script.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.mozilla.javascript.*;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Setter
@Getter
@Slf4j
public class JavaScriptService extends AbstractScriptService {

    public static final String PATTERN = "```javascript\\s*([\\s\\S]*?)```";

    public static final String PREFIX = "Error: ";

    protected ExecutorService executorService;

    @Override
    public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
        String jsScript = StringUtils.defaultString(this.extract(script), JsonUtils.clean(script));
        if (log.isInfoEnabled()) {
            log.info("JavaScriptService script={}", jsScript);
        }
        Assert.hasText(jsScript, "JavaScriptService can not be empty: " + script);
        try {
            return StringUtils.defaultIfEmpty(this.executorService.submit(new JsFuture(scriptEnv, jsScript)).get(timeout, TimeUnit.MILLISECONDS), "");
        } catch (Exception e) {
            if (e.getCause() != null) {
                if (JavaScriptException.class.isAssignableFrom(e.getCause().getClass())) {
                    // JS异常处理
                    JavaScriptException javaScriptException = JavaScriptException.class.cast(e.getCause());
                    String message = javaScriptException.getMessage();
                    if (javaScriptException.getValue() != null) {
                        message = javaScriptException.getValue().toString().startsWith(JavaScriptService.PREFIX) ? javaScriptException.getValue().toString().replaceFirst(JavaScriptService.PREFIX, "") : javaScriptException.getValue().toString();
                    }
                    if (log.isDebugEnabled()) {
                        log.debug("JavaScriptService exception={}-{}", jsScript, message);
                    }
                    Assert.hasText(message, "JavaScriptService exception can not be empty");
                    throw new JavaScriptException(message, javaScriptException.sourceName(), javaScriptException.lineNumber());
                } else if (EcmaError.class.isAssignableFrom(e.getCause().getClass())) {
                    throw new JavaScriptException(EcmaError.class.cast(e.getCause()).getMessage());
                }
            }
            throw e;
        }
    }

    protected String getObject(Object object) {
        if (NativeObject.class.isAssignableFrom(object.getClass())) {
            NativeObject nativeObject = NativeObject.class.cast(object);
            StringBuffer buffer = new StringBuffer("{");
            Object[] ids = nativeObject.getIds();
            for (int i = 0; i < ids.length; i++) {
                Object id = ids[i];
                Object value = nativeObject.get(id.toString(), nativeObject);
                if (i > 0) {
                    buffer.append(", ");
                }
                buffer.append("\"").append(id).append("\": ");
                if (String.class.isAssignableFrom(value.getClass())) {
                    buffer.append("\"").append(value).append("\"");
                } else {
                    buffer.append(this.getObject(value));
                }
            }
            buffer.append("}");
            return buffer.toString();
        }
        if (Undefined.class.isAssignableFrom(object.getClass())) {
            throw new WorkflowException("Undefined", ProtocolCode.C500);
        } else {
            return object.toString();
        }
    }

    // 提取JS
    protected String extract(String script) {
        Matcher matcher = Pattern.compile(JavaScriptService.PATTERN).matcher(script);
        if (matcher.find()) {
            return matcher.group(1).trim();
        }
        return null;
    }

    public class ConsoleError extends BaseFunction {
        private final StringBuffer buffer = new StringBuffer();

        @Override
        public Object call(Context cx, Scriptable scope, Scriptable thisObj, Object[] args) {
            for (Object arg : args) {
                this.buffer.append(JavaScriptService.this.getObject(arg));
            }
            throw new WorkflowException(this.buffer.toString());
        }
    }

    public class ConsoleLog extends BaseFunction {
        private final StringBuffer buffer = new StringBuffer();

        @Override
        public Object call(Context cx, Scriptable scope, Scriptable thisObj, Object[] args) {
            for (Object arg : args) {
                this.buffer.append(JavaScriptService.this.getObject(arg));
            }
            return this.buffer.toString();
        }
    }

    public class JsFuture implements Callable<String> {

        protected final ScriptEnv scriptEnv;

        protected final String script;

        public JsFuture(ScriptEnv scriptEnv, String script) {
            this.scriptEnv = scriptEnv;
            this.script = script;
        }

        @Override
        public String call() throws Exception {
            try {
                Context context = Context.enter();
                return JavaScriptService.this.getObject(context.evaluateString(this.getScriptableObject(context), this.script, "JavaScriptCode", 1, null));
            } finally {
                Context.exit();
            }
        }

        public ScriptableObject getScriptableObject(Context context) {
            ScriptableObject scriptableObject = context.initStandardObjects();
            for (String key : this.scriptEnv.keySet()) {
                scriptableObject.put(key, scriptableObject, this.scriptEnv.get(key));
            }
            Scriptable console = context.newObject(scriptableObject);
            console.put("error", console, new ConsoleError());
            console.put("log", console, new ConsoleLog());
            console.setParentScope(scriptableObject);
            console.setPrototype(ScriptableObject.getClassPrototype(scriptableObject, "Object"));
            scriptableObject.put("console", scriptableObject, console);
            return scriptableObject;
        }
    }

    @ConditionalOnProperty(name = "javascript.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ScriptInitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Bean
        @ConditionalOnMissingBean(value = JavaScriptService.class)
        public JavaScriptService javaScriptService() throws Exception {
            JavaScriptService javaScriptService = new JavaScriptService();
            BeanUtils.copyProperties(this, javaScriptService);
            log.info("JavaScriptService inited");
            return javaScriptService;
        }
    }
}