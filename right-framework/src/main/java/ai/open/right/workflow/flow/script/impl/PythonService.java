package ai.open.right.workflow.flow.script.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Slf4j
@Setter
@Getter
public class PythonService extends AbstractScriptService {

    public static final String PYTHON_HOME = System.getProperty("PYTHON_HOME", System.getenv("PYTHON_HOME"));

    public static final String PATTERN = "```python\\s*([\\s\\S]*?)```";

    public static final String PREFIX = "```python";

    public static final String SUFFIX = "```";

    // 操作系统兼容
    private Boolean transfer = System.getProperty("os.name").toLowerCase().contains("win");

    // Output分段获取（用于处理大批量结果返回）
    protected Integer segment;

    // Python执行文件
    protected String python;

    // Python安装路径
    protected String home;

    // 执行用户路径，默认${user.dir}
    protected String path;

    @PostConstruct
    public void init() {
        this.home = StringUtils.defaultIfBlank(this.home, PythonService.PYTHON_HOME);
    }

    public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
        Process process = null;
        BufferedReader stdInput = null;
        BufferedReader stdError = null;
        String pyScript = StringUtils.defaultIfBlank(this.extract(script), this.clean(script));
        if (log.isDebugEnabled()) {
            log.debug("PythonService content={}", pyScript);
        }
        Assert.hasText(pyScript, "PythonService script can not be empty: " + script);
        try {
            // 用户执行路径
            String path = StringUtils.defaultIfEmpty(this.path, System.getProperty("user.dir"));
            if (log.isDebugEnabled()) {
                log.debug("PythonService script path={},script={}", path, pyScript);
            }
            ProcessBuilder processBuilder = new ProcessBuilder(this.home + File.separator + this.python, "-X", "utf8", "-", path);
            if (!CollectionUtils.isEmpty(scriptEnv)) {
                processBuilder.environment().putAll(scriptEnv);
            }
            process = processBuilder.start();
            try (OutputStream os = process.getOutputStream()) {
                os.write(this.buildScript(pyScript).getBytes(StandardCharsets.UTF_8));
                os.flush();
            }
            stdInput = new BufferedReader(new InputStreamReader(process.getInputStream(), StandardCharsets.UTF_8));
            stdError = new BufferedReader(new InputStreamReader(process.getErrorStream(), StandardCharsets.UTF_8));
            StringWriter inputWriter = new StringWriter();
            int totalTime = 0;
            while (totalTime < timeout) {
                // 至少为 1
                int segmentTime = Math.max(timeout / this.segment, 1);
                if (process.waitFor(segmentTime, TimeUnit.MILLISECONDS)) {
                    int exitValue = process.exitValue();
                    if (exitValue == 0) {
                        IOUtils.copy(stdInput, inputWriter);
                        return StringUtils.defaultIfEmpty(inputWriter.toString(), "");
                    } else {
                        this.buildError(stdError);
                    }
                } else {
                    if (process.getInputStream().available() > 0) {
                        IOUtils.copy(stdInput, inputWriter);
                    }
                }
                totalTime += segmentTime;
            }
            throw new WorkflowException("PythonService script was timed out", ProtocolCode.C502);
        } finally {
            IOUtils.closeQuietly(stdInput);
            IOUtils.closeQuietly(stdError);
            if (process != null) {
                process.destroyForcibly();
            }
        }
    }

    protected void buildError(BufferedReader stdError) throws IOException {
        throw new WorkflowException(IOUtils.toString(stdError));
    }

    protected String buildScript(String script) {
        return "import sys; __file__ = sys.argv[1];" + System.lineSeparator() + script;
    }

    protected String extract(String script) {
        Matcher matcher = Pattern.compile(PythonService.PATTERN).matcher(script);
        if (matcher.find()) {
            String group = matcher.group(1).trim();
            return this.transfer ? group.replaceAll("\"", "\\\\\"") : group;
        }
        return null;
    }

    protected String clean(String script) {
        script = script.trim();
        script = this.transfer ? script.replaceAll("\"", "\\\\\"") : script;
        if (script.startsWith(PythonService.PREFIX) && script.endsWith(PythonService.SUFFIX)) {
            script = new StringBuffer(script).delete(script.length() - PythonService.SUFFIX.length(), script.length()).delete(0, PythonService.PREFIX.length()).toString();
        }
        if (log.isDebugEnabled()) {
            log.debug("PythonService script after cleaning={}", script);
        }
        Assert.hasText(script, "PythonService script can not be empty");
        return script;
    }

    @ConditionalOnProperty(name = "python.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ScriptInitConfig {

        @Value("${python.transfer:}")
        // 操作系统兼容
        private Boolean transfer = System.getProperty("os.name").toLowerCase().contains("win");

        @Value("${python.segment:10}")
        // Output分段获取（用于处理大批量结果返回）
        protected Integer segment;

        @Value("${python.version:python3}")
        // Python执行文件
        protected String python;

        @Value("${python.home:}")
        // Python安装路径
        protected String home;

        @Value("${python.path:}")
        // 执行用户路径，默认${user.dir}
        protected String path;

        @Bean
        @ConditionalOnMissingBean(value = PythonService.class)
        public PythonService pythonService() throws Exception {
            PythonService pythonService = new PythonService();
            BeanUtils.copyProperties(this, pythonService);
            log.info("PythonService inited, transfer={},segment={},python={},home={},path={}", pythonService.getTransfer(), pythonService.getSegment(), pythonService.getPython(), pythonService.getHome(), pythonService.getPath());
            return pythonService;
        }
    }
}
