package ai.open.right.workflow.flow.track.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.track.TrackDimension;
import ai.open.right.workflow.flow.track.TrackFunCall;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

@Slf4j
@Setter
@Getter
public class RedisTrackFunCallService implements TrackFunCallService {

    private static final List<TrackFunCall> EMPTY = Collections.unmodifiableList(new ArrayList<TrackFunCall>());

    protected RedisTemplate<String, Object> redis4funCall;

    // Redis版本
    protected Boolean version6_2_0 = false;

    // SECONDS
    // Track持久化时间
    protected Integer expire;

    @Override
    public List<TrackFunCall> restore(TrackDimension trackDimension) throws Exception {
        try {
            Assert.notNull(this.redis4funCall, "Redis4funCall can not be empty");
            String key = this.getKey(trackDimension);
            List<Object> data = this.fetchData(key);
            if (log.isDebugEnabled()) {
                log.debug("Track fun call restore: key={}, size={}", key, CollectionUtils.isEmpty(data) ? 0 : data.size());
            }
            if (CollectionUtils.isEmpty(data)) {
                return RedisTrackFunCallService.EMPTY;
            }
            List<TrackFunCall> trackFunCalls = new ArrayList<TrackFunCall>();
            for (Object d : data) {
                try {
                    trackFunCalls.add(JsonUtils.read(GzipUtils.decompress((byte[]) d), TrackFunCall.class));
                } catch (Exception e) {
                    log.warn("Track fun call failed while serializing ( " + key + "): " + e.getMessage(), e);
                }
            }
            return trackFunCalls;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return RedisTrackFunCallService.EMPTY;
        }
    }

    @Override
    public void store(TrackFunCall trackFunCall) throws Exception {
        try {
            Assert.notNull(this.redis4funCall, "Redis4funCall can not be empty");
            String key = this.getKey(trackFunCall.getTrackDimension());
            byte[] kBytes = key.getBytes(StandardCharsets.UTF_8);
            byte[] vBytes = GzipUtils.compress(JsonUtils.write(trackFunCall).getBytes(StandardCharsets.UTF_8));
            List<Object> result = this.redis4funCall.executePipelined(new RedisTrackSetCallback(kBytes, vBytes, this.expire));
            if (log.isDebugEnabled()) {
                log.debug("Track fun call store: key={},result={}", key, result);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected String getKey(TrackDimension trackDimension) {
        return RedisConfig.DOMAIN + RedisTrackFunCallService.class.getSimpleName() + trackDimension.getBiz() + trackDimension.getChat() + trackDimension.getDevice() + trackDimension.getTrack();
    }

    protected List<Object> fetchData(String key) {
        if (this.version6_2_0) {
            if (log.isDebugEnabled()) {
                log.debug("Trace fun call restore using redis batch");
            }
            return this.redis4funCall.opsForList().rightPop(key, Integer.MAX_VALUE);
        } else {
            if (log.isDebugEnabled()) {
                log.debug("Trace fun call restore using redis each");
            }
            List<Object> data = new ArrayList<Object>();
            Object each = null;
            while ((each = this.redis4funCall.opsForList().rightPop(key)) != null) {
                data.add(each);
            }
            return data;
        }
    }

    @ConditionalOnProperty(name = "track.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected RedisTemplate<String, Object> redis4funCall;

        @Value("${track.funcall.version6_2_0:true}")
        // Redis版本
        protected Boolean version6_2_0 = false;

        // SECONDS
        @Value("${track.funcall.expire:3600000}")
        // Track持久化时间
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = TrackFunCallService.class)
        public TrackFunCallService trackFunCallService() throws Exception {
            RedisTrackFunCallService trackFunCallService = new RedisTrackFunCallService();
            BeanUtils.copyProperties(this, trackFunCallService);
            log.info("RedisTrackFunCallService inited: version6_2_0={},expire={}", trackFunCallService.getVersion6_2_0(), trackFunCallService.getExpire());
            return trackFunCallService;
        }
    }
}
