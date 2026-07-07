package ai.deepright.cli.function;

import ai.deepright.cli.*;
import ai.deepright.cli.insert.CliInsertService;
import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.module.HttpProtocol;
import ai.deepright.router.RouterDevice;
import ai.deepright.router.RouterService;
import ai.deepright.safety.SafetyService;
import ai.deepright.utils.TemplateChecker;
import ai.deepright.workflow.worktask.MarkQueryWorkTask;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.resouce.ResourceService;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SpinExec;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.sync.SyncConfig;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.FilenameUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.dao.QueryTimeoutException;
import org.springframework.data.redis.RedisSystemException;
import org.springframework.data.redis.core.RedisOperations;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.SessionCallback;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;
import java.util.*;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
// CLI生成指定队列
public class CliSubFunction extends BaseFunction implements CliSubFetcher, CliTransfer {

    public static final String LANG_KEY_GENERAL_MESSAGE = "cli.general.message";

    public static final String LANG_KEY_LOGIC_MESSAGE = "cli.logic.message";

    public static final String NAME = "fun_cli_sub";

    public static final String KEY = "cli@sub";

    protected RedisTemplate<String, Object> redis4event;

    protected CloseableHttpAsyncClient resource;

    protected CliInsertService cliInsertService;

    protected ResourceService resourceService;

    protected RouterService routerService;

    protected CliSubBlocker cliSubBlocker;

    protected SafetyService safetyService;

    protected HttpProtocol httpProtocol;

    protected SysStore sysStore;

    protected String template4safety;

    protected Integer timeout4cmd;

    // 软检查超时
    protected Integer timeout4ops;

    // 队列最大等待超时
    protected Integer timeout4sub;

    // 自旋间隔，用于动态计算总次数，默认2s
    protected Integer interval;

    protected Integer oversize;

    protected Integer minimum;

    // 队列Key过期时间（ms），与Pub共享
    protected Integer expire;

    // 是否软检查
    protected Boolean safety;

    protected Boolean debug;

    // 异步返回时的固定文案（模型需要使用）
    protected String def;

    @PostConstruct
    public void init() throws Exception {
        // IOUtils/JsonUtils负责关闭资源
        this.template4safety = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4safety).openStream()), StandardCharsets.UTF_8);
        // 覆盖（rewrite），不需要重入
        // 启动检测，必要资源
        Assert.hasText(this.template4safety, "The template safety must not be empty");
    }

    public Object call(FunctionContext functionContext) throws Exception {
        return this.call(functionContext.getWorkTask(), true, false);
    }

    public Object call(FunctionContext functionContext, Boolean waitPub) throws Exception {
        return this.call(functionContext.getWorkTask(), waitPub, false);
    }

    // exempted=true, 软检查豁免
    public Object call(WorkflowTask workTask, Boolean waitPub) throws Exception {
        return this.call(workTask, waitPub, false);
    }

    // exempted=true, 软检查豁免
    public Object call(WorkflowTask workTask, Boolean waitPub, Boolean exempted) throws Exception {
        try {
            CliPubSub.checkValid(workTask);
            String query = StringUtils.trim(workTask.printQuery().getQuery());
            query = JsonUtils.like(query) ? query : JsonUtils.extract(query);
            Assert.isTrue(JsonUtils.like(query), "The cli request cannot be parsed as JSON due to unexpected formatting: " + workTask.getQuery());
            Map<String, Object> source = JsonUtils.read(query, Map.class);
            RouterDevice router = this.buildTargetDevice(workTask, source);
            Assert.notEmpty(source, "The cli@sub can not be empty: " + router.key());
            Integer timeout = this.buildTimeout(workTask, source);
            if (log.isInfoEnabled()) {
                log.info("The cli@sub router timeout={}, key={}", timeout, router.key());
            }
            Boolean unwind = MapUtils.getBoolean(source, "unwind", true);
            CliSubOps cliSubOps = this.buildSubOps(workTask, source, exempted);
            CliSubData subData = CliSubData.builder()
                    .agentId(StringUtils.defaultIfEmpty(MapUtils.getString(source, "target_agent"), FeatureUtils.buildAgentId(workTask)))
                    // 文件后缀名，如果执行指令则为cmd（代码触发），Fun Call都为fun（模型触发）
                    .type(StringUtils.defaultIfEmpty(MapUtils.getString(source, "type"), unwind ? CliPubSub.FUN : CliPubSub.CMD))
                    .suffix(StringUtils.defaultIfEmpty(MapUtils.getString(source, "suffix"), cliSubOps.getApps()))
                    .why(StringUtils.defaultIfEmpty(MapUtils.getString(source, "why_do_this"), ""))
                    .workspace(FeatureUtils.buildWorkspace(workTask))
                    .cmd(MapUtils.getString(source, CliPubSub.CMD))
                    .conversation(workTask.getConversation())
                    // 超时（秒），转换为毫秒
                    .ddl(this.buildDDL(workTask, timeout))
                    .tid(UUID.randomUUID().toString())
                    .chat(workTask.getChat())
                    .router(router.key())
                    .subOps(cliSubOps)
                    .timeout(timeout)
                    // LLM FunCall默认unwind=true，编码调用默认false
                    .unwind(unwind)
                    .build().check();
            // 在线状态检测，操作软检查
            this.checkHeartbeat(workTask, router, subData, source, exempted);
            this.checkSafetyCmd(workTask, router, subData, source, exempted);
            this.source(workTask, router, subData.getWhy(), false);
            this.source(workTask, router, subData.getCmd(), true);
            this.logic(workTask, router, subData.getWhy());
            // 推送CLI任务队列到指定设备 并推送到端
            Object subRequest = new CliSubRequestExec(this.redis4event, this.interval, timeout, this.expire, router.getDevice(), JsonUtils.write(subData)).exec();
            // 需要模型可读
            Assert.notNull(subRequest, "The response is timeout, please try a different command.");
            if (waitPub) {
                // 等待通道结果
                Object subResponse = new CliSubResponseExec(this.redis4event, this.interval, timeout, subData.getTid()).exec();
                byte[] result = byte[].class.cast(subResponse);
                this.checkResponse(workTask, timeout, result);
                // GZIP+BASE64后的CMD
                String content = new String(GzipUtils.decompressAsBase64(new String(result, StandardCharsets.UTF_8)), StandardCharsets.UTF_8);
                CliPubData pubData = JsonUtils.read(content, CliPubData.class);
                this.insert(workTask, pubData);
                // CLI成功，清除BLOCK计数
                this.cliSubBlocker.clean(workTask);
                this.notify(workTask, router);
                // 失败回显
                if (!pubData.isOk()) {
                    this.source(workTask, router, pubData.getCmd(), true);
                }
                // FunCall报文自动拆箱一定是base64解码，不需要二次解码
                // 调用异常（如超时）则结果不为JSON
                if (subData.getUnwind() && JsonUtils.like(content)) {
                    // 强制拆包
                    return pubData.forceText(this.resource, this.sysStore, this.timeout4cmd, false).getCmd();
                } else {
                    return content;
                }
            } else {
                // 异步，固定结果
                return this.def;
            }
        } catch (RedisSystemException | QueryTimeoutException e) {
            this.cliSubBlocker.block(workTask);
            throw new WorkflowException(e, this.debug ? ProtocolCode.C500 : ProtocolCode.C915).needSilent();
        } catch (Exception e) {
            this.cliSubBlocker.block(workTask);
            throw e;
        }
    }

    @Override
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, Boolean waitPub, String suffix, String device, String agent, String cmd, String why) throws Exception {
        Map<String, Object> cli = new HashMap<String, Object>();
        // 使用当前WorkTask Name
        cli.put("target_device", device);
        cli.put("target_agent", agent);
        cli.put("why_do_this", why);
        cli.put("sub_ops", subOps);
        cli.put("suffix", suffix);
        cli.put("unwind", false);
        cli.put("cmd", cmd);
        Object result = this.call(new MarkQueryWorkTask(workTask, JsonUtils.write(cli)), waitPub, subOps.isExempted());
        // 是否使用结果
        if (waitPub) {
            return JsonUtils.transfer(result, CliPubData.class);
        } else {
            return CliPubData.builder()
                    .cmd(String.valueOf(result))
                    .status(CliPubData.SUCCESS)
                    .encode(CliPubData.TEXT)
                    .build();
        }
    }

    @Override
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, Boolean waitPub, String suffix, String device, String cmd, String why) throws Exception {
        return this.command(workTask, subOps, waitPub, suffix, device, "", cmd, why);
    }

    @Override
    public CliPubData command(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, Boolean waitPub, String cmd, String why) throws Exception {
        return this.command(workTask, subOps, waitPub, "", router.getDevice(), router.getAgent(), cmd, why);
    }

    @Override
    public CliPubData command(WorkflowTask workTask, RouterDevice router, Boolean waitPub, String suffix, String cmd, String why) throws Exception {
        return this.command(workTask, CliSubOps.builder().build(), waitPub, suffix, router.getDevice(), router.getAgent(), cmd, why);
    }

    @Override
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, Boolean waitPub, String device, String cmd, String why) throws Exception {
        return this.command(workTask, subOps, waitPub, CliPubSub.CMD, device, "", cmd, why);
    }

    @Override
    public CliPubData command(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, String cmd, String why) throws Exception {
        return this.command(workTask, subOps, true, CliPubSub.CMD, router.getDevice(), router.getAgent(), cmd, why);
    }

    @Override
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, String device, String cmd, String why) throws Exception {
        return this.command(workTask, subOps, true, CliPubSub.CMD, device, "", cmd, why);
    }

    @Override
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, String cmd, String why) throws Exception {
        return this.command(workTask, subOps, true, CliPubSub.CMD, workTask.getDevice(), "", cmd, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String suffix, String device, String agent, String path, String why) throws Exception {
        // file:///则去除file://
        path = FeatureUtils.escapePath(workTask, FeatureUtils.escapeFile(workTask, path));
        return this.command(workTask, subOps == null ? CliSubOps.builder()
                .app(List.of("cat"))
                .r(List.of(path))
                .build() : subOps, true, StringUtils.isEmpty(suffix) ? FilenameUtils.getExtension(Paths.get(path).getFileName().toString()) : suffix, device, agent, new StringBuffer("cat ").append(FeatureUtils.escapeShell(workTask, path)).toString(), why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String suffix, String device, String agent, List<String> paths, String why) throws Exception {
        List<String> normalized = this.normalizePaths(workTask, paths);
        return this.command(workTask, subOps == null ? CliSubOps.builder()
                .app(List.of("cat"))
                .r(normalized)
                .build() : subOps, true, this.buildFetchSuffix(suffix, normalized), device, agent, this.buildFetchCmd(workTask, normalized), why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String suffix, String device, String path, String why) throws Exception {
        return this.fetch(workTask, subOps, suffix, device, "", path, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, String path, String why) throws Exception {
        return this.fetch(workTask, subOps, "", router.getDevice(), router.getAgent(), path, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, List<String> paths, String why) throws Exception {
        return this.fetch(workTask, subOps, "", router.getDevice(), router.getAgent(), paths, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String device, String path, String why) throws Exception {
        return this.fetch(workTask, subOps, "", device, "", path, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, String path, String why) throws Exception {
        return this.fetch(workTask, null, "", router.getDevice(), router.getAgent(), path, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, List<String> paths, String why) throws Exception {
        return this.fetch(workTask, null, "", router.getDevice(), router.getAgent(), paths, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, String device, String path, String why) throws Exception {
        return this.fetch(workTask, null, "", device, "", path, why);
    }

    @Override
    public CliPubData fetch(WorkflowTask workTask, String path, String why) throws Exception {
        return this.fetch(workTask, null, "", workTask.getDevice(), "", path, why);
    }

    @Override
    public CliTransferData transfer(WorkflowTask workTask, RouterDevice source, RouterDevice target, String path, String why) throws Exception {
        // 读取并推送
        CliPubData sourceData = this.fetch(workTask, source, path, why);
        Assert.isTrue(sourceData.isOk(), sourceData.getCmd());
        String targetFile = FeatureUtils.escapePath(FeatureFlag.isWindows(target.getSys()), target.getWorkspace() + File.separator + "tmp" + File.separator + FilenameUtils.getName(path));
        String command = StringUtils.equalsIgnoreCase(CliPubData.URL, sourceData.getEncode()) ? CliPubSub.buildPushURL(workTask, this.httpProtocol.dataHost(sourceData.getCmd()), targetFile) : CliPubSub.buildPushCmd(workTask, this.sysStore, this.oversize, sourceData.getCmd(), targetFile);
        CliPubData targetData = this.command(workTask, CliSubOps.builder()
                .app(List.of("cat", "curl", "mkdir"))
                .w(List.of(targetFile))
                .build(), true, FilenameUtils.getExtension(Paths.get(path).getFileName().toString()), target.getDevice(), target.getAgent(), command, why);
        Assert.isTrue(targetData.isOk(), targetData.getCmd());
        return CliTransferData.builder()
                .targetPubData(targetData)
                .sourcePubData(sourceData)
                .target(targetFile)
                .source(path)
                .build();
    }

    protected List<String> normalizePaths(WorkflowTask workTask, List<String> paths) throws Exception {
        Assert.notEmpty(paths, "The fetch paths can not be empty");
        List<String> normalized = new ArrayList<String>(paths.size());
        for (String path : paths) {
            Assert.hasText(path, "The fetch paths can not contain empty values");
            normalized.add(FeatureUtils.escapePath(workTask, FeatureUtils.escapeFile(workTask, path)));
        }
        return normalized;
    }

    protected String buildFetchCmd(WorkflowTask workTask, List<String> paths) throws Exception {
        Assert.notEmpty(paths, "The fetch paths can not be empty");
        List<String> escaped = new ArrayList<String>(paths.size());
        for (String path : paths) {
            escaped.add(FeatureUtils.escapeShell(workTask, path));
        }
        return "cat " + StringUtils.join(escaped, " ");
    }

    protected String buildFetchSuffix(String suffix, List<String> paths) throws Exception {
        if (!StringUtils.isEmpty(suffix)) {
            return suffix;
        }
        if (paths.size() != 1) {
            return "";
        }
        return FilenameUtils.getExtension(Paths.get(paths.get(0)).getFileName().toString());
    }

    // LLM安全检查
    protected String buildSafetyQuery(WorkflowTask workTask, CliSubData subData, CliSubOps subOps, String safety) throws Exception {
        String query = this.template4safety.replace("#write", subOps.hasW() ? ArrayUtils.toString(subOps.getW()) : "");
        // 精确匹配
        query = query.replace("#app", subOps.hasApp() ? ArrayUtils.toString(subOps.getApp()) : "");
        query = query.replace("#read", subOps.hasR() ? ArrayUtils.toString(subOps.getR()) : "");
        query = query.replace("#why", subData.getWhy());
        query = query.replace("#rules", safety);
        if (log.isWarnEnabled() && !TemplateChecker.check(query)) {
            log.warn("The query template contains unexpected characters; please check: {}", query);
        }
        return query;
    }

    // 同时检查是否都为绝对路径
    protected CliSubOps buildSubOps(WorkflowTask workTask, Map<String, Object> source, Boolean exempted) throws Exception {
        try {
            CliSubOps cliSubOps = JsonUtils.transfer(MapUtils.getObject(source, "sub_ops"), CliSubOps.class);
            cliSubOps = cliSubOps != null ? cliSubOps : CliSubOps.builder().build();
            cliSubOps.setExempted(exempted);
            return cliSubOps.reConfig(workTask);
        } catch (MismatchedInputException | JsonParseException e) {
            if (exempted) {
                // 降级
                if (log.isWarnEnabled()) {
                    log.warn("The JSON parsing failed: unrecognized `CliSubOps`");
                }
                return CliSubOps.builder().exempted(true).build();
            } else {
                throw e;
            }
        }
    }

    protected RouterDevice buildTargetDevice(WorkflowTask workTask, Map<String, Object> source) throws Exception {
        String device = StringUtils.defaultIfEmpty(MapUtils.getString(source, "target_device"), workTask.getDevice());
        return new RouterDevice(workTask, device, StringUtils.defaultIfEmpty(MapUtils.getString(source, "target_agent"), StringUtils.equalsIgnoreCase(device, workTask.getDevice()) ? FeatureUtils.buildAgentId(workTask) : ""));
    }

    protected Integer buildTimeout(WorkflowTask workTask, Map<String, Object> source) throws Exception {
        Integer timeout = this.timeout4sub;
        Integer seconds = MapUtils.getInteger(source, "timeout_seconds");
        if (seconds != null && seconds > 0) {
            timeout = Math.min(Math.max(this.minimum, seconds), this.timeout4sub);
        }
        return (int) TimeUnit.MILLISECONDS.convert(timeout, TimeUnit.SECONDS);
    }

    protected Long buildDDL(WorkflowTask workTask, Integer timeout) {
        return System.currentTimeMillis() + timeout;
    }

    // 软安全检查
    protected void checkSafetyCmd(WorkflowTask workTask, RouterDevice routerDevice, CliSubData subData, Map<String, Object> source, Boolean exempted) throws Exception {
        if (!exempted) {
            // 获取软检查的Prompt
            String safety = this.safetyService.safety(workTask);
            if (this.safety && !StringUtils.isEmpty(safety)) {
                SyncConfig syncConfig = SyncConfig.builder()
                        .reQuery(this.buildSafetyQuery(workTask, subData, subData.getSubOps(), safety))
                        .timeout(this.timeout4ops)
                        .workflow("cli@ops")
                        .workTask(workTask)
                        .build();
                Map<String, Object> result = JsonUtils.read(this.localhost(workTask, syncConfig).get(), Map.class);
                Assert.isTrue(MapUtils.getBooleanValue(result, "decision", false), "The command can not be allowed: " + MapUtils.getString(result, "why_do_this"));
            }
        }
    }

    protected void checkHeartbeat(WorkflowTask workTask, RouterDevice routerDevice, CliSubData subData, Map<String, Object> source, Boolean exempted) throws Exception {
        if (!routerDevice.isSame(workTask)) {
            Assert.isTrue(this.routerService.hasHeartbeat(routerDevice), "The device [" + routerDevice.getDevice() + "][" + routerDevice.getAgent() + "]'s heartbeat was not detected.");
        }
    }

    protected void checkResponse(WorkflowTask workTask, Integer timeout, byte[] result) throws Exception {
        // 用于提示模型的错误
        if (ArrayUtils.isEmpty(result)) {
            throw new WorkflowException("The request has timed out after " + timeout + " ms, please ensure the command is valid.").needSilent();
        }
    }

    // 推端消息
    protected void source(WorkflowTask workTask, RouterDevice targetDevice, String content, Boolean cmd) throws Exception {
        if (targetDevice.isLoop(workTask) && !FeatureFlag.isSilent(workTask) && !StringUtils.isEmpty(content)) {
            // Cmd在终端需要特殊展示，用于标记（parsed.biz === 'cli' && parsed.workflow === 'sub'）
            if (cmd) {
                this.source(workTask, CliSubFunction.KEY, CliPrinter.code(StringUtils.trim(content)));
            } else {
                this.source(workTask, ProviderRequestService.KEY_INTERNAL + workTask.getWorkflow(), CliPrinter.format(content, CliPrinter.SIZE_N));
            }
        }
    }

    // LogicVerifyAssistant特殊处理
    protected void logic(WorkflowTask workTask, RouterDevice targetDevice, String content) throws Exception {
        if (FeatureFlag.isLogic(workTask) && !StringUtils.isEmpty(content)) {
            this.source(workTask, CliPrinter.process(FeatureField.KEY_LOGIC), XmlResourceLang.get(CliSubFunction.LANG_KEY_LOGIC_MESSAGE).replace("#content", content));
        }
    }

    protected void notify(WorkflowTask workTask, RouterDevice targetDevice) throws Exception {
        if (targetDevice.isLoop(workTask) && !FeatureFlag.isSilent(workTask)) {
            this.source(workTask, CliPrinter.process(CliSubFunction.NAME), XmlResourceLang.get(CliSubFunction.LANG_KEY_GENERAL_MESSAGE));
        }
    }

    protected void insert(WorkflowTask workTask, CliPubData pubData) throws Exception {
        if (pubData.hasInsert()) {
            this.cliInsertService.insert(workTask, pubData.getInsert());
        }
    }

    public static class CliSubRequestCallable implements SessionCallback<Object> {

        protected final Integer expire;

        protected final String device;

        protected final String data;

        public CliSubRequestCallable(Integer expire, String device, String data) throws Exception {
            this.device = CliSubFetcher.getDeviceKey(device);
            this.expire = expire;
            this.data = data;
        }

        @Override
        @SuppressWarnings("unchecked")
        public Object execute(RedisOperations operations) {
            // 推送CLI任务队列到指定设备 并推送到端
            Object result = operations.opsForList().rightPush(this.device, this.data.getBytes(StandardCharsets.UTF_8));
            operations.expire(this.device, this.expire, TimeUnit.MILLISECONDS);
            return result;
        }
    }

    public static class CliSubResponseExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4event;

        protected final Integer interval;

        protected final String tid;

        public CliSubResponseExec(RedisTemplate<String, Object> redis4event, Integer interval, Integer timeout, String tid) throws Exception {
            super(timeout, (int) Math.ceil((double) timeout / interval));
            this.tid = CliSubFetcher.getTidKey(tid);
            this.redis4event = redis4event;
            this.interval = interval;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4event.opsForList().leftPop(this.tid, this.interval, TimeUnit.MILLISECONDS);
            } catch (RedisSystemException e) {
                if (log.isInfoEnabled()) {
                    log.info(e.getMessage());
                }
                return null;
            } catch (Exception e) {
                WorkflowException.dolog(e);
                return null;
            }
        }
    }

    public static class CliSubRequestExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4event;

        protected final Integer expire;

        protected final String device;

        protected final String data;

        public CliSubRequestExec(RedisTemplate<String, Object> redis4event, Integer interval, Integer timeout, Integer expire, String device, String data) throws Exception {
            super(timeout, (int) Math.ceil((double) timeout / interval));
            this.redis4event = redis4event;
            this.device = device;
            this.expire = expire;
            this.data = data;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4event.execute(new CliSubRequestCallable(this.expire, this.device, this.data));
            } catch (Exception e) {
                WorkflowException.dolog(e);
                return null;
            }
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected RedisTemplate<String, Object> redis4event;

        @Autowired
        protected CloseableHttpAsyncClient resource;

        @Autowired
        protected CliInsertService cliInsertService;

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected RouterService routerService;

        @Autowired
        protected CliSubBlocker cliSubBlocker;

        @Autowired
        protected SafetyService safetyService;

        @Autowired
        protected HttpProtocol httpProtocol;

        @Autowired
        protected SysStore sysStore;

        @Value("${cli.sub.template.safety:classpath:config/cli/safety.md}")
        protected String template4safety;

        // 恢复资源的超时
        @Value("${cli.cmd.timeout:30000}")
        protected Integer timeout4cmd;

        // 安全检查的超时
        @Value("${cli.ops.timeout:120000}")
        protected Integer timeout4ops;

        // 等待Pub队列的时间（秒，需要与LLM描述统一）
        @Value("${cli.sub.timeout:300}")
        protected Integer timeout4sub;

        @Value("${cli.sub.interval:2000}")
        protected Integer interval;

        // 超过这个字节大小的Funcall要阻止
        @Value("${cli.sub.oversize:51200}")
        protected Integer oversize;

        // 最小超时时间
        @Value("${cli.sub.minimum:60}")
        protected Integer minimum;

        // 与Pub共享的队列超时
        @Value("${cli.expire:300000}")
        protected Integer expire;

        // 是否使用软件检查
        @Value("${cli.safety:false}")
        protected Boolean safety;

        @Value("${debug:false}")
        protected Boolean debug;

        // 不等待结果时的返回
        @Value("${cli.def:SUCCESS}")
        protected String def;

        @Bean(CliSubFunction.NAME)
        @ConditionalOnMissingBean(name = CliSubFunction.NAME)
        public CliSubFunction cliSubFunction() throws Exception {
            CliSubFunction cliSubFunction = new CliSubFunction();
            BeanUtils.copyProperties(this, cliSubFunction);
            log.info("CliSubFunction inited");
            return cliSubFunction;
        }
    }
}
