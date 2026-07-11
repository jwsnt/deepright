package ai.deepright.cli.function;

import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.router.RouterDevice;
import ai.deepright.router.RouterService;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SpinExec;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.BaseFunction;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisTemplate;

import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
// CLI获取队列任务，提交客户端信息
public class CliGetFunction extends BaseFunction {

    public static final String NAME = "fun_cli_get";

    // 事件类使用redis4event;
    protected RedisTemplate<String, Object> redis4event;

    protected RouterService routerService;

    // 队列LeftPop的超时
    protected Integer interval;

    // 自旋总超时，默认2s
    protected Integer timeout;

    protected Integer display;

    // 自旋总次数
    protected Integer circle;

    protected Boolean debug;

    public Object call(FunctionContext functionContext) throws Exception {
        try {
            CliPubSub.checkValid(functionContext.getWorkTask());
            RouterDevice router = new RouterDevice(functionContext.getWorkTask());
            if (log.isInfoEnabled()) {
                log.info("The cli@get router key={}", router.key());
            }
            // 没有获取到可以用Result或超时则返回
            Object rest = null;
            long start = System.currentTimeMillis();
            long close = 0;
            while (rest == null && ((this.timeout - this.interval) > close)) {
                rest = new CliGetExec(this.redis4event, this.interval, this.timeout, this.circle, router.getDevice(), this.debug).exec();
                if (rest != null) {
                    // CMD + TID
                    CliSubData subData = JsonUtils.read((byte[]) rest, CliSubData.class).check();
                    if (!subData.isExpired()) {
                        if (log.isInfoEnabled()) {
                            log.info("The cli@get fetch the task, key={}", router.key());
                        }
                        rest = subData;
                    } else if (log.isWarnEnabled()) {
                        // 过期丢弃
                        log.warn("The cli@get task was created={}, expired={}, timeout={}, cmd={}, why={}, key={}", subData.getCreated(), subData.getDdl(), subData.getTimeout(), StringUtils.abbreviate(subData.getCmd(), this.display), StringUtils.abbreviate(subData.getWhy(), this.display), router.key());
                        rest = null;
                    }
                }
                close = System.currentTimeMillis() - start;
            }
            // 超时日志
            if (log.isWarnEnabled() && close > (this.timeout + this.interval)) {
                log.warn("The cli@get fetch the value, key={}, value={}, waiting={}", router.key(), rest, (System.currentTimeMillis() - functionContext.getWorkTask().getCreated()));
            }
            return rest;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return null;
        } finally {
            // 最后同步心跳
            this.heartbeat(functionContext.getWorkTask());
        }
    }

    // 心跳
    protected void heartbeat(WorkflowTask workTask) throws Exception {
        this.routerService.heartbeat(workTask);
    }

    public static class CliGetExec extends SpinExec {

        // Event专用Redis
        protected final RedisTemplate<String, Object> redis4event;

        protected final Integer interval;

        protected final String device;

        protected final Boolean debug;

        public CliGetExec(RedisTemplate<String, Object> redis4event, Integer interval, Integer timeout, Integer circle, String device, Boolean debug) throws Exception {
            // Timeout内尝试Circle次
            super(timeout, circle);
            // Pub/Sub使用Device对齐
            this.device = CliSubFetcher.getDeviceKey(device);
            this.redis4event = redis4event;
            // 如果通道为空，堵塞的时间
            this.interval = interval;
            this.debug = debug;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4event.opsForList().leftPop(this.device, this.interval, TimeUnit.MILLISECONDS);
            } catch (Exception e) {
                if (this.debug) {
                    log.error(e.getMessage(), e);
                }
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
        protected RouterService routerService;

        @Value("${cli.get.interval:1000}")
        protected Integer interval;

        @Value("${cli.get.timeout:15000}")
        protected Integer timeout;

        @Value("${cli.get.display:500}")
        protected Integer display;

        @Value("${cli.get.circle:10}")
        protected Integer circle;

        @Value("${debug:false}")
        protected Boolean debug;

        // 从Redis获取指定设备任务，每interval取一次，直到timeout，循环尝试timeout/circle次
        @Bean(CliGetFunction.NAME)
        @ConditionalOnMissingBean(name = CliGetFunction.NAME)
        public CliGetFunction cliGetFunction() throws Exception {
            CliGetFunction cliGetFunction = new CliGetFunction();
            BeanUtils.copyProperties(this, cliGetFunction);
            log.info("CliGetFunction inited");
            return cliGetFunction;
        }
    }
}
