package ai.open.right.workflow.flow.block.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.block.BlockService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.TimeUnit;

@Slf4j
@Setter
@Getter
public class RedisBlockServiceImpl implements BlockService {

    protected RedisTemplate<String, Object> redis4array;

    // SECONDS
    // 缓存时间
    protected Integer expire;

    @Override
    public void block(String biz, String chat, String device, WorkflowTask workTask) throws Exception {
        try {
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(biz, chat, device);
            byte[] val = (byte[]) this.redis4array.opsForValue().get(key);
            if (log.isDebugEnabled()) {
                log.debug("Redis block key={}", key);
            }
            if (val == null) {
                return;
            }
            long remain = Long.parseLong(new String(val, StandardCharsets.UTF_8)) - workTask.getCreated();
            if (remain > 0) {
                throw new WorkflowException("The workflow is blocked, remain=" + remain, ProtocolCode.C0);
            }
        } catch (WorkflowException e) {
            throw e;
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }


    @Override
    public void block(String biz, String chat, WorkflowTask workTask) throws Exception {
        this.block(biz, chat, workTask.getDevice(), workTask);
    }

    @Override
    public void block(WorkflowTask workTask) throws Exception {
        this.block(workTask.getBiz(), workTask.getChat(), workTask);
    }

    @Override
    public void submit(String biz, String chat, String device, WorkflowTask workTask, Long timestamp) throws Exception {
        try {
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(biz, chat, device);
            if (log.isDebugEnabled()) {
                log.debug("Redis block submit key={}, timestamp={}", key, timestamp);
            }
            this.redis4array.opsForValue().set(key, String.valueOf(timestamp).getBytes(StandardCharsets.UTF_8), this.expire, TimeUnit.SECONDS);
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    @Override
    public void submit(String biz, String chat, WorkflowTask workTask, Long timestamp) throws Exception {
        this.submit(biz, chat, workTask.getDevice(), workTask, timestamp);
    }

    @Override
    public void submit(String biz, String chat, String device, WorkflowTask workTask) throws Exception {
        this.submit(biz, chat, device, workTask, this.getLastTime(workTask));
    }

    @Override
    public void submit(String biz, String chat, WorkflowTask workTask) throws Exception {
        this.submit(biz, chat, workTask, this.getLastTime(workTask));
    }

    @Override
    public void submit(WorkflowTask workTask, Long timestamp) throws Exception {
        this.submit(workTask.getBiz(), workTask.getChat(), workTask, timestamp);
    }

    @Override
    public void submit(WorkflowTask workTask) throws Exception {
        this.submit(workTask.getBiz(), workTask.getChat(), workTask);
    }

    protected String getKey(String biz, String chat, String device) {
        return RedisConfig.DOMAIN + RedisBlockServiceImpl.class.getSimpleName() + biz + chat + device;
    }

    protected Long getLastTime(WorkflowTask workTask) {
        // 不能Trim，防止破坏MarkDown
        String val = StringUtils.defaultIfEmpty(workTask.getQuery(), "");
        if (!StringUtils.isNumeric(val)) {
            return workTask.getCreated();
        } else {
            return Long.parseLong(val);
        }
    }

    @ConditionalOnProperty(name = "block.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected RedisTemplate<String, Object> redis4array;

        // SECONDS
        @Value("${block.expire:300}")
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = BlockService.class)
        public BlockService blockService() throws Exception {
            RedisBlockServiceImpl blockService = new RedisBlockServiceImpl();
            BeanUtils.copyProperties(this, blockService);
            log.info("RedisBlockServiceImpl inited");
            return blockService;
        }
    }
}
