package ai.deepright.cli.function;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.utils.TemplateChecker;
import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import com.fasterxml.jackson.annotation.JsonProperty;
import jakarta.annotation.PostConstruct;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
public class CliVerifyFunction extends BaseFunction {

    public static final String NAME = "fun_cli_verify";

    public static final String AGREE = "agree";

    public static final String YES = "__YES__";

    public static final String NO = "__NO__";

    protected ExecutorService executorService;

    protected ResourceService resourceService;

    protected CliSubFetcher cliSubFetcher;

    protected String template;

    protected Integer timeout;

    protected Boolean debug;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template), "The template must not be empty");
    }

    @Override
    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = functionContext.getWorkTask().printQuery();
        String query = StringUtils.trim(workTask.getQuery());
        query = JsonUtils.like(query) ? query : JsonUtils.extract(query);
        WorkflowException.checkCondition(!(JsonUtils.like(query)), "The cli verify cannot be parsed as JSON due to unexpected formatting: " + workTask.getQuery());
        Verifier verifier = JsonUtils.read(query, Verifier.class);
        return this.checkArtifacts(workTask, verifier);
    }

    // 检查文件状态
    protected String checkArtifacts(WorkflowTask workTask, Verifier verifier) throws Exception {
        if (verifier.hasArtifact()) {
            List<Future<VerifierResult>> futures = new ArrayList<Future<VerifierResult>>();
            // 检查文件是否存在
            for (Artifact artifact : verifier.getArtifact()) {
                futures.add(this.executorService.submit(VerifierCallable.builder()
                        .cliSubFetcher(this.cliSubFetcher)
                        .workTask(workTask)
                        .artifact(artifact)
                        .build()));
            }
            StringBuffer buffer = new StringBuffer();
            for (int index = 0; index < verifier.getArtifact().size(); index++) {
                String resource = verifier.getArtifact().get(index).getResource();
                Future<VerifierResult> future = futures.get(index);
                try {
                    VerifierResult verifierResult = future.get(this.timeout, TimeUnit.MILLISECONDS);
                    if (!verifierResult.getExists()) {
                        buffer.append(resource).append(": not exists").append(System.lineSeparator());
                    }
                } catch (Exception e) {
                    buffer.append(resource).append(": ").append(e.getMessage()).append(System.lineSeparator());
                    future.cancel(true);
                    if (this.debug) {
                        WorkflowException.dolog(e);
                    }
                }
            }
            if (!StringUtils.isEmpty(buffer)) {
                // 异常错误
                return this.buildException(workTask, buffer);
            }
        }
        // 正常返回
        return CliVerifyFunction.AGREE;
    }

    protected String buildException(WorkflowTask workTask, StringBuffer content) throws Exception {
        // 用于提示模型
        String exception = this.template.replace("#content", content.toString()).replace("#workspace", FeatureUtils.buildWorkspace(workTask));
        if (log.isWarnEnabled() && !TemplateChecker.check(exception)) {
            log.warn("The verify template contains unexpected characters; please check: {}", exception);
        }
        return exception;
    }

    @Builder
    @Getter
    public static class VerifierCallable implements Callable<VerifierResult> {

        protected CliSubFetcher cliSubFetcher;

        protected WorkflowTask workTask;

        protected Artifact artifact;

        @Override
        public VerifierResult call() throws Exception {
            String cmd = this.buildPushExists(this.workTask, this.artifact.getResource(), CliVerifyFunction.YES, CliVerifyFunction.NO);
            CliPubData pubData = this.cliSubFetcher.command(this.workTask, CliSubOps.builder()
                    .r(List.of(this.artifact.getResource()))
                    .app(List.of("curl"))
                    .exempted(true)
                    .build(), cmd, "");
            return VerifierResult.builder()
                    // 检查是否存在
                    .exists(StringUtils.containsIgnoreCase(StringUtils.trim(pubData.valid().getCmd()), CliVerifyFunction.YES))
                    .cmd(cmd)
                    .build();
        }

        // 构建检查指令
        protected String buildPushExists(WorkflowTask workTask, String url, String yes, String no) throws Exception {
            StringBuffer cmd = new StringBuffer();
            // 网络或file:///开头就保持原样，否则加上前缀
            // 统一为 file:/// 开头：
            // ///Users/a.png -> file:///Users/a.png
            // file:/Users/a.png -> file:///Users/a.png
            // file:///Users/a.png -> file:///Users/a.png
            // file:////Users/a.png -> file:///Users/a.png
            url = !MediaTransferUtils.isNetwork(url) ? "file:///" + url.replaceFirst("(?i)^file:/*", "").replaceFirst("^/+", "").replace(" ", "%20") : url;
            // curl -s -I "路径或URL" > nul && echo 存在 1 || echo 不存在 0
            cmd.append("curl -s -I ").append(FeatureUtils.escapeShell(workTask, url)).append(" && echo ").append(FeatureUtils.escapeShell(workTask, yes)).append("|| echo ").append(FeatureUtils.escapeShell(workTask, no));
            return cmd.toString();
        }
    }

    @Getter
    @Builder
    public static class VerifierResult {

        protected Boolean exists;

        protected String cmd;
    }

    @Getter
    @Setter
    public static class Verifier {

        protected List<Artifact> artifact;

        @JsonProperty("why_do_this")
        protected String why;

        public Boolean hasArtifact() {
            return !CollectionUtils.isEmpty(this.artifact);
        }
    }

    @Getter
    @Setter
    public static class Artifact {

        protected String resource;

        protected String function;
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Value("${cli.verify.template:classpath:config/cli/verify.md}")
        protected String template;

        @Value("${cli.verify.timeout:30000}")
        protected Integer timeout;

        @Value("${sys.debug:false}")
        protected Boolean debug;

        @Bean(CliVerifyFunction.NAME)
        @ConditionalOnMissingBean(name = CliVerifyFunction.NAME)
        public CliVerifyFunction cliVerifyFunction() throws Exception {
            CliVerifyFunction verifyFunction = new CliVerifyFunction();
            BeanUtils.copyProperties(this, verifyFunction);
            log.info("CliVerifyFunction inited");
            return verifyFunction;
        }
    }
}
