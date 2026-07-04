package ai.deepright.cli;

import ai.open.right.WorkflowException;
import ai.open.right.utils.SpinExec;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.block.BlockService;
import ai.deepright.router.RouterDevice;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisOperations;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.SessionCallback;
import org.springframework.util.Assert;

import java.util.List;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
// CLI连续失败时终止
public class CliSubBlocker {

    public static final String KEY_PREFIX = "cli:block:";

    public static final String NAME = "cliSubBlocker";

    protected RedisTemplate<String, Object> redis4array;

    protected BlockService blockService;

    // 自旋总超时，默认2s
    protected Integer timeout;

    // 自旋总次数
    protected Integer circle;

    // Block计数Key的缓存，默认30s
    protected Integer expire;

    // Block触发次数
    protected Integer times;

    public void block(WorkflowTask workTask) throws Exception {
        workTask.markQuery();
        try {
            // Incr（计数）
            Object result = new CliBlockExec(this.redis4array, this.expire, this.timeout, this.circle, this.getKey(workTask)).exec();
            Assert.notNull(result, "The cli blocker incr can not null");
            // 触发次数
            int current = Number.class.cast(List.class.cast(result).getFirst()).intValue();
            if (current >= this.times) {
                if (log.isWarnEnabled()) {
                    log.warn("The cli blocker will be started, device={}, chat={}", workTask.getDevice(), workTask.getChat());
                }
                // 终止当前时间前的所有任务（workTask.resetQuery()）
                workTask.setQuery(String.valueOf(System.currentTimeMillis()));
                // @See ReConfigWorkflow, Block React，成功后清除计数
                this.blockService.submit("main", workTask.getChat(), RouterDevice.key(workTask), workTask);
                // 清除计数
                this.clean(workTask);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        } finally {
            workTask.resetQuery();
        }
    }

    public void clean(WorkflowTask workTask) throws Exception {
        try {
            Object rest = new CliCleanExec(this.redis4array, this.timeout, this.circle, this.getKey(workTask)).exec();
            Assert.notNull(rest, "The cli blocker can not be cleared");
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    // 设备+会话维度
    protected String getKey(WorkflowTask workTask) throws Exception {
        return CliSubBlocker.KEY_PREFIX + RouterDevice.key(workTask) + workTask.getChat();
    }

    public static class CliBlockCallback implements SessionCallback<Object> {

        protected final Integer expire;

        protected final String key;

        public CliBlockCallback(String key, Integer expire) {
            this.expire = expire;
            this.key = key;
        }

        @Override
        @SuppressWarnings("unchecked")
        public Object execute(RedisOperations operations) {
            operations.opsForValue().increment(this.key);
            operations.expire(this.key, this.expire, TimeUnit.MILLISECONDS);
            return null;
        }
    }

    public static class CliBlockExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4array;

        protected final Integer expire;

        protected final String key;

        public CliBlockExec(RedisTemplate<String, Object> redis4array, Integer expire, Integer timeout, Integer circle, String key) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.expire = expire;
            this.key = key;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4array.executePipelined(new CliBlockCallback(this.key, this.expire));
            } catch (Exception e) {
                log.error(e.getMessage(), e);
                return null;
            }
        }
    }

    public static class CliCleanExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4array;

        protected final String key;

        public CliCleanExec(RedisTemplate<String, Object> redis4array, Integer timeout, Integer circle, String key) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.key = key;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4array.delete(this.key);
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
        protected RedisTemplate<String, Object> redis4array;

        @Autowired
        protected BlockService blockService;

        @Value("${cli.block.timeout:5000}")
        protected Integer timeout;

        @Value("${cli.block.circle:5}")
        protected Integer circle;

        @Value("${cli.block.expire:30000}")
        protected Integer expire;

        @Value("${cli.block.times:3}")
        protected Integer times;

        // 单次Loop出现times次中断后，尝试在Timeout时间内最多尝试5次来终止正在执行任务（Timestamp前），成功后终止expire秒
        @Bean(CliSubBlocker.NAME)
        @ConditionalOnMissingBean(name = CliSubBlocker.NAME)
        public CliSubBlocker cliSubBlocker() throws Exception {
            CliSubBlocker cliSubBlocker = new CliSubBlocker();
            BeanUtils.copyProperties(this, cliSubBlocker);
            log.info("CliSubBlocker inited");
            return cliSubBlocker;
        }
    }
}
