package ai.deepright.cli.function;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.router.RouterDevice;
import ai.open.right.utils.*;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.DefStore;
import ai.open.right.workflow.flow.file.FileStore;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisOperations;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.SessionCallback;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
// CLI提交指定队列
public class CliPubFunction extends BaseFunction {

    public static final String NAME = "fun_cli_pub";

    // 事件类使用redis4event;
    protected RedisTemplate<String, Object> redis4event;

    protected CloseableHttpAsyncClient resource;

    protected CliSubFetcher cliSubFetcher;

    protected DefStore defStore;

    // 超过指定大小需要转存（数据不进Redis，默认1.5M）
    protected Integer oversize;

    // 自旋总超时，默认10s
    protected Integer timeout;

    // 自旋总次数
    protected Integer circle;

    // 队列Key过期时间（ms），与Sub共享
    protected Integer expire;

    protected String store;

    public Object call(FunctionContext functionContext) throws Exception {
        WorkflowTask workTask = log.isDebugEnabled() ? functionContext.getWorkTask().printQuery() : functionContext.getWorkTask();
        CliPubSub.checkValid(workTask);
        RouterDevice router = new RouterDevice(workTask);
        if (log.isInfoEnabled()) {
            log.info("The cli@pub router key={}", router.key());
        }
        CliPubData pubData = workTask.getObjectQuery(CliPubData.class);
        WorkflowException.check(pubData == null, "The cli@pub can not be empty: " + router.key(), ProtocolCode.C400);
        // 解码
        pubData = this.decodeData(workTask, pubData.check());
        // GZIP+BASE64后的CMD
        Object result = new CliPubExec(this.redis4event, this.timeout, this.circle, this.expire, GzipUtils.compressAsBase64(JsonUtils.write(pubData)), pubData.getTid()).exec();
        WorkflowException.check(result == null, "The cli@pub failed: " + pubData.getTid(), ProtocolCode.C400);
        return result;
    }

    protected CliPubData decodeData(WorkflowTask workTask, CliPubData pubData) throws Exception {
        // OOM防护由Netty控制
        byte[] decode = GzipUtils.decompressAsBase64(pubData.getCmd());
        // 执行成功、Type!=FUN 且超过指定大小或二进制格式（图片等），上传URL
        if (pubData.isOk() && !pubData.isType(CliPubSub.FUN) && (BytesUtils.utf8Bytes(pubData.getCmd()) >= this.oversize || SuffixUtils.isBinary(pubData.getSuffix()))) {
            FileStore fileStore = this.defStore.fetchStore(this.store);
            WorkflowException.check(fileStore == null, "The file store can not be empty", ProtocolCode.C400);
            pubData.setCmd(fileStore.store(decode, pubData.getSuffix(), workTask));
            pubData.setEncode(CliPubData.URL);
            return pubData;
        } else {
            if (pubData.isType(CliPubSub.FUN) && BytesUtils.utf8Bytes(pubData.getCmd()) >= this.oversize) {
                // 如果是Type=Fun，校验大小防止LLM溢出
                // 丢弃原内容
                pubData.setCmd("The response size exceeds the maximum limit of [" + this.oversize + "], please use a different command.");
            } else {
                // 当!pubData.isOk()且!isType(FUN)且decode超大时，依旧原样返回（不会进模型）
                pubData.setCmd(new String(decode, StandardCharsets.UTF_8));
            }
            pubData.setEncode(CliPubData.TEXT);
            return pubData;
        }
    }

    public static class CliPubCallback implements SessionCallback<Object> {

        private final Integer expire;

        private final String data;

        private final String tid;

        public CliPubCallback(Integer expire, String data, String tid) throws Exception {
            this.tid = CliSubFetcher.getTidKey(tid);
            this.expire = expire;
            this.data = data;
        }

        @Override
        @SuppressWarnings("unchecked")
        public Object execute(RedisOperations operations) {
            // 推送通道并指定超时
            Object result = operations.opsForList().rightPush(this.tid, this.data.getBytes(StandardCharsets.UTF_8));
            operations.expire(this.tid, this.expire, TimeUnit.MILLISECONDS);
            return result;
        }
    }

    public static class CliPubExec extends SpinExec {

        // Event专用Redis
        private final RedisTemplate<String, Object> redis4event;

        private final Integer expire;

        private final String data;

        private final String tid;

        public CliPubExec(RedisTemplate<String, Object> redis4event, Integer timeout, Integer circle, Integer expire, String data, String tid) {
            // Timeout内尝试Circle次
            super(timeout, circle);
            this.redis4event = redis4event;
            this.expire = expire;
            this.data = data;
            this.tid = tid;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4event.execute(new CliPubCallback(this.expire, this.data, this.tid));
            } catch (Exception e) {
                log.error(e.getMessage(), e);
                return null;
            }
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class CliInitConfig {

        @Autowired
        protected RedisTemplate<String, Object> redis4event;

        @Autowired
        protected CloseableHttpAsyncClient resource;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected DefStore defStore;

        // 超过这个字节大小使用Cli URL协议
        @Value("${cli.pub.oversize:204800}")
        protected Integer oversize;

        // 10秒内尝试10次
        @Value("${cli.pub.timeout:10000}")
        protected Integer timeout;

        @Value("${cli.pub.circle:10}")
        protected Integer circle;

        // Pub Sub共享过期时间
        @Value("${cli.expire:300000}")
        protected Integer expire;

        // 使用的转存服务
        @Value("${cli.store:file.store.sys}")
        protected String store;

        @Bean(CliPubFunction.NAME)
        @ConditionalOnMissingBean(name = CliPubFunction.NAME)
        public CliPubFunction cliPubFunction() throws Exception {
            CliPubFunction cliPubFunction = new CliPubFunction();
            BeanUtils.copyProperties(this, cliPubFunction);
            log.info("CliPubFunction inited");
            return cliPubFunction;
        }
    }
}
