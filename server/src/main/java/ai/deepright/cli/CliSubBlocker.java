package ai.deepright.cli;

import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.block.BlockService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
@Getter
@Setter
// CLI连续失败时终止
public class CliSubBlocker {

    public static final String KEY_PREFIX = "cli:block:";

    public static final String NAME = "cliSubBlocker";

    protected BlockService blockService;

    // Block触发次数
    protected Integer times;

    public void block(WorkflowTask workTask) throws Exception {
        // Incr（计数）
        Integer current = MapUtils.getInteger(workTask.getUserContext().getMetadata(), CliSubBlocker.KEY_PREFIX, 0);
        try {
            if (current >= this.times) {
                if (log.isWarnEnabled()) {
                    log.warn("The cli blocker will be started, device={}, chat={}", workTask.getDevice(), workTask.getChat());
                }
                // @See ReConfigWorkflow, Block React，成功后清除计数
                this.blockService.submit("main", workTask.getChat(), RouterDevice.key(workTask), workTask, System.currentTimeMillis());
                // 清除计数
                this.clean(workTask);
            } else {
                this.incr(workTask, current);
            }
        } catch (Exception e) {
            this.incr(workTask, current);
            WorkflowException.dolog(e);
        }
    }

    public void clean(WorkflowTask workTask) throws Exception {
        workTask.getUserContext().getMetadata().remove(CliSubBlocker.KEY_PREFIX);
    }

    protected void incr(WorkflowTask workTask, Integer current) {
        workTask.getUserContext().putMetadata(CliSubBlocker.KEY_PREFIX, current + 1);
    }

    @Configuration
    @Getter
    @Setter
    public static class CliInitConfig {

        @Autowired
        protected BlockService blockService;

        @Value("${cli.block.times:50}")
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
