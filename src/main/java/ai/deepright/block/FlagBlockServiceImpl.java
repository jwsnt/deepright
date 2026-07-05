package ai.deepright.block;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.block.BlockService;
import ai.open.right.workflow.flow.block.impl.RedisBlockServiceImpl;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class FlagBlockServiceImpl extends RedisBlockServiceImpl {

    public static final String NAME = "flag_block";

    public void submit(String biz, String chat, String device, WorkflowTask workTask, Long timestamp) throws Exception {
        super.submit(biz, chat, device, workTask, timestamp);
        // 标记需要检查，减少redis读
        workTask.getUserContext().putMetadata(FlagBlockServiceImpl.NAME, true);
    }

    public void block(String biz, String chat, String device, WorkflowTask workTask) throws Exception {
        // 检查标记检查
        if (MapUtils.getBoolean(workTask.getUserContext().getMetadata(), FlagBlockServiceImpl.NAME, false)) {
            super.block(biz, chat, device, workTask);
            // 如果开启标记且未被堵塞则清除标记
            if (workTask.getUserContext().delMetadata(FlagBlockServiceImpl.NAME, Boolean.class) && log.isInfoEnabled()) {
                log.info("The workflow is unblocked");
            }
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class FlagInitConfig extends InitConfig {

        @Override
        @Bean(FlagBlockServiceImpl.NAME)
        @ConditionalOnMissingBean(BlockService.class)
        public BlockService blockService() throws Exception {
            FlagBlockServiceImpl blockServiceImpl = new FlagBlockServiceImpl();
            BeanUtils.copyProperties(this, blockServiceImpl);
            log.info("FlagBlockServiceImpl inited");
            return blockServiceImpl;
        }
    }
}
