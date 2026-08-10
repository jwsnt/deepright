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
import java.util.Arrays;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class CommandService extends AbstractScriptService {

    public static final String COMMAND_HOME = System.getProperty("COMMAND_HOME", System.getenv("COMMAND_HOME"));

    public static final String SPLIT_SPACE = " ";

    public static final String PREFIX = "```";

    public static final String SUFFIX = "```";

    // Output分段获取（用于处理大批量结果返回）
    protected Integer segment;

    // Command安装路径
    protected String home;

    @PostConstruct
    public void init() {
        this.home = StringUtils.defaultIfBlank(this.home, CommandService.COMMAND_HOME);
    }

    public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
        Process process = null;
        BufferedReader stdInput = null;
        BufferedReader stdError = null;
        if (log.isDebugEnabled()) {
            log.debug("CommandService content={}", script);
        }
        Assert.hasText(script, "CommandService script can not be empty: " + script);
        try {
            String[] part = StringUtils.split(this.clean(script), CommandService.SPLIT_SPACE);
            if (!StringUtils.isEmpty(this.home)) {
                // 如果指定了Home则重写Command路径
                part[0] = this.home + File.separator + part[0];
            }
            part = Arrays.stream(part).map(String::trim).toArray(String[]::new);
            if (log.isInfoEnabled()) {
                log.info("CommandService script={}", Arrays.toString(part));
            }
            ProcessBuilder processBuilder = new ProcessBuilder(part);
            if (!CollectionUtils.isEmpty(scriptEnv)) {
                processBuilder.environment().putAll(scriptEnv);
            }
            process = processBuilder.start();
            stdInput = new BufferedReader(new InputStreamReader(process.getInputStream(), StandardCharsets.UTF_8));
            stdError = new BufferedReader(new InputStreamReader(process.getErrorStream(), StandardCharsets.UTF_8));
            StringWriter inputWriter = new StringWriter();
            int totalTime = 0;
            while (totalTime < timeout) {
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
            throw new WorkflowException("CommandService script was timed out", ProtocolCode.C502);
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

    protected String clean(String script) {
        script = script.trim();
        if (script.startsWith(CommandService.PREFIX) && script.endsWith(CommandService.SUFFIX)) {
            script = new StringBuffer(script)
                    .delete(script.length() - CommandService.SUFFIX.length(), script.length())
                    .delete(0, CommandService.PREFIX.length()).toString();
        }
        if (log.isDebugEnabled()) {
            log.debug("CommandService script after cleaning={}", script);
        }
        Assert.hasText(script, "CommandService script can not be empty");
        return script;
    }

    @ConditionalOnProperty(name = "cmd.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ScriptInitConfig {

        @Value("${cmd.segment:10}")
        // Output分段获取（用于处理大批量结果返回）
        protected Integer segment;

        @Value("${cmd.home:}")
        // Command安装路径
        protected String home;

        @Bean
        @ConditionalOnMissingBean(value = CommandService.class)
        public CommandService commandService() throws Exception {
            CommandService commandLineService = new CommandService();
            BeanUtils.copyProperties(this, commandLineService);
            log.info("CommandService inited, segment={}, home={}", commandLineService.getSegment(), commandLineService.getHome());
            return commandLineService;
        }
    }
}
