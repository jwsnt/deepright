package ai.open.right.workflow.flow.llm.store.history.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;
import org.springframework.util.CollectionUtils;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Set;

@Slf4j
@Setter
@Getter
/**
 * score = -created（毫秒），数值越大（越接近 0）表示消息越旧，越负表示越新
 * desc（与可选 now 组合）：
 * - desc=false, now=null：全集中按 score 正序 取前 nums 条，语义上取 最新 一批，返回前会 reverse，时间线需结合调用方理解
 * - desc=false, now!=null：只在 score 开区间 (now, 0]（实现为 min=now+1）内按 正序 取前 nums 条，在 早于边界 now 对应时刻的消息里取 最新nums 条，不含 score=now
 * - desc=true, now=null：全集中按 score 逆序 取前 nums 条，语义上取 最旧 一批，返回前会 reverse
 * - desc=true, now!=null：只在 (now, 0] 内按 逆序 取前 nums 条，在 早于边界 的消息里取 最旧 nums 条，不含 score=now
 *
 * now（[now,0]）控制区间
 * desc 控制取值方向
 * num 控制取值数量
 */
public class RedisHistoryStore extends BaseHistoryStore implements HistoryStore {

    private static final List<History> EMPTY = Collections.unmodifiableList(new ArrayList<History>());

    protected RedisTemplate<String, Object> redis4array;

    protected Integer maxsize;

    // SECONDS
    protected Integer expire;

    @Override
    // 如果当前时间是T，end传入为-(T-90秒)，desc=false，取过去 90 秒到现在的数据
    public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now, Long end) throws Exception {
        try {
            String key = this.getKey(dimension, scene);
            ZSetOperations<String, Object> zSetOps = this.redis4array.opsForZSet();
            int limit = Math.min(nums, this.maxsize);
            Set<Object> pairData = null;
            if (desc) {
                // 最旧N：score越大越旧，取score>now（不召回 score=now），即严格早于now的消息
                // 存储：score=-created（时间戳取负）。时间戳越大越新，但-时间戳数值越小（如-200 < -100），故score越大（越接近 0）= 越旧，score越小（越负）= 越新
                if (now != null) {
                    // 逆序：分数区间 (now, now + offset]，用 now+1 实现开区间，排除 score=now
                    pairData = zSetOps.reverseRangeByScore(key, now + 1, end, 0, limit);
                } else {
                    pairData = zSetOps.reverseRange(key, end, limit);
                }
            } else {
                // 最新N：在早于now的消息里取score最小的limit条，不召回score=now
                if (now != null) {
                    // 正序：分数区间 (now, 0]，用 now+1 实现开区间，排除 score=now
                    pairData = zSetOps.rangeByScore(key, now + 1, end, 0, limit);
                } else {
                    pairData = zSetOps.range(key, end, limit);
                }
            }
            if (log.isDebugEnabled()) {
                log.debug("History data restore from redis: key={}, now={}, desc={}, history_size={}", key, now, desc, CollectionUtils.isEmpty(pairData) ? 0 : pairData.size());
            }
            if (CollectionUtils.isEmpty(pairData)) {
                return RedisHistoryStore.EMPTY;
            }
            List<History> histories = new ArrayList<>();
            for (Object each : pairData) {
                try {
                    HistoryPair historyPair = this.restore(dimension, JsonUtils.read(GzipUtils.decompress((byte[]) each), HistoryPair.class));
                    if (historyPair != null) {
                        History[] pairs = historyPair.buildHistories();
                        History answer = pairs[1];
                        History query = pairs[0];
                        if (answer != null) {
                            answer = this.preRestore(dimension, answer, scene);
                            answer.setReference(History.REFERENCE_SERVER);
                            histories.add(answer);
                        }
                        if (query != null) {
                            query = this.preRestore(dimension, query, scene);
                            query.setReference(History.REFERENCE_SERVER);
                            histories.add(query);
                        }
                    }
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            // 排序（按插入时间）
            if (CollectionUtils.isEmpty(histories)) {
                return RedisHistoryStore.EMPTY;
            } else {
                Collections.reverse(histories);
                return histories;
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return RedisHistoryStore.EMPTY;
        }
    }

    @Override
    // 如果指定now则为开区间(now,0]
    public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now) throws Exception {
        return this.restore(dimension, scene, nums, desc, now, 0L);
    }

    @Override
    public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc) throws Exception {
        return this.restore(dimension, scene, nums, desc, null);
    }

    @Override
    public List<History> restore(Dimension dimension, String scene, Integer nums, Long now) throws Exception {
        return this.restore(dimension, scene, nums, false, now);
    }

    @Override
    public List<History> restore(Dimension dimension, String scene, Integer nums) throws Exception {
        return this.restore(dimension, scene, nums, false, null);
    }

    @Override
    public void store(Dimension dimension, List<String> repositories, String query, String answer, String reasoning, Integer expire, Integer nums, Long now) throws Exception {
        HistoryPair pair = new HistoryPair();
        pair.setApi(ProviderRequest.REQUEST_DEF);
        pair.setModel(ProviderRequestModel.DEF);
        pair.setReasoning(reasoning);
        pair.setAnswer(answer);
        pair.setQuery(query);
        pair.setCreated(now);
        this.store(dimension, repositories, pair, expire, nums);
    }

    @Override
    public void store(Dimension dimension, List<String> repositories, String query, String answer, Integer expire, Integer nums, Long now) throws Exception {
        HistoryPair pair = new HistoryPair();
        // 使用默认
        pair.setApi(ProviderRequest.REQUEST_DEF);
        pair.setModel(ProviderRequestModel.DEF);
        pair.setAnswer(answer);
        pair.setQuery(query);
        pair.setCreated(now);
        this.store(dimension, repositories, pair, expire, nums);
    }

    @Override
    public void store(Dimension dimension, List<String> repositories, List<HistoryPair> pairs, Integer expire, Integer nums) throws Exception {
        try {
            List<RedisPair> store = new ArrayList<RedisPair>();
            for (HistoryPair each : pairs) {
                // 都为空
                if (StringUtils.isEmpty(each.getAnswer()) && StringUtils.isEmpty(each.getQuery())) {
                    continue;
                }
                if ((each = this.store(dimension, each)) == null) {
                    continue;
                }
                List<byte[]> kBytes = new ArrayList<byte[]>();
                for (String repository : repositories) {
                    String key = this.getKey(dimension, repository);
                    kBytes.add(key.getBytes(StandardCharsets.UTF_8));
                }
                byte[] history = this.getVal(dimension, each);
                store.add(RedisPair.builder()
                        .created(each.getCreated())
                        .history(history)
                        .keys(kBytes)
                        .build());
            }
            if (!CollectionUtils.isEmpty(store)) {
                List<Object> result = this.redis4array.executePipelined((new RedisStoreCallback(store, expire != null ? expire : this.expire, Math.min(nums, this.maxsize))));
                if (log.isDebugEnabled()) {
                    log.debug("History data store to redis: store={}, result={}", store.size(), result);
                }
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {
        try {
            // 都为空
            if (StringUtils.isEmpty(pair.getQuery()) && StringUtils.isEmpty(pair.getAnswer())) {
                return;
            }
            if ((pair = this.store(dimension, pair)) == null) {
                return;
            }
            List<byte[]> kBytes = new ArrayList<byte[]>();
            List<String> keys = new ArrayList<String>();
            for (String repository : repositories) {
                String key = this.getKey(dimension, repository);
                kBytes.add(key.getBytes(StandardCharsets.UTF_8));
                keys.add(key);
            }
            // 最终存入的时间为-now（越新的消息，数值越小）
            byte[] history = this.getVal(dimension, pair);
            List<Object> result = this.redis4array.executePipelined((new RedisStoreCallback(kBytes, history, expire != null ? expire : this.expire, Math.min(nums, this.maxsize), pair.getCreated())));
            if (log.isDebugEnabled()) {
                log.debug("History data store to redis: key={}, result={}", keys, result);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    // 时间为-now（越新的消息，数值越小）
    public void clear(Dimension dimension, List<String> repositories, Boolean desc, Long now) throws Exception {
        try {
            List<byte[]> kBytes = new ArrayList<byte[]>();
            List<String> keys = new ArrayList<String>();
            for (String each : repositories) {
                String key = this.getKey(dimension, each);
                kBytes.add(key.getBytes(StandardCharsets.UTF_8));
                keys.add(key);
            }
            List<Object> result = this.redis4array.executePipelined(new RedisClearCallback(kBytes, desc, now));
            if (log.isDebugEnabled()) {
                log.debug("History data clear: key={}, result={} ", keys, result);
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    // 时间为-now（越新的消息，数值越小）
    public void clear(Dimension dimension, List<String> repositories, Long now) throws Exception {
        this.clear(dimension, repositories, false, now);
    }

    // 子类覆盖
    protected History preRestore(Dimension dimension, History history, String scene) throws Exception {
        return history;
    }

    protected byte[] getVal(Dimension dimension, HistoryPair pair) throws Exception {
        return GzipUtils.compress(JsonUtils.write(pair));
    }

    protected String getKey(Dimension dimension, String scene) throws Exception {
        String[] pair = SplitUtils.split(scene, dimension.getBiz());
        return RedisConfig.DOMAIN + RedisHistoryStore.class.getSimpleName() + pair[0] + dimension.getChat() + pair[1] + dimension.getDevice();
    }

    public static class RedisStoreCallback implements RedisCallback<Void> {

        protected final List<RedisPair> pairs;

        protected final Integer expire;

        protected final Integer num;

        public RedisStoreCallback(List<byte[]> keys, byte[] history, Integer expire, Integer num, Long now) {
            this(List.of(RedisPair.builder()
                    .history(history)
                    .keys(keys)
                    .created(now)
                    .build()), expire, num);
        }

        public RedisStoreCallback(List<RedisPair> pairs, Integer expire, Integer num) {
            this.expire = expire;
            this.pairs = pairs;
            this.num = num;
        }

        @Override
        public Void doInRedis(RedisConnection connection) throws DataAccessException {
            for (RedisPair pair : this.pairs) {
                for (byte[] kBytes : pair.getKeys()) {
                    connection.zAdd(kBytes, -pair.getCreated(), pair.getHistory());
                    // 删除
                    connection.zSetCommands().zRemRange(kBytes, this.num, -1);
                    connection.keyCommands().expire(kBytes, this.expire);
                }
            }
            return null;
        }
    }

    public static class RedisClearCallback implements RedisCallback<Void> {

        protected final List<byte[]> keys;

        protected final Boolean desc;

        protected final Long now;

        public RedisClearCallback(List<byte[]> keys, Boolean desc, Long now) {
            this.keys = keys;
            this.desc = desc;
            this.now = now;
        }

        @Override
        public Void doInRedis(RedisConnection connection) throws DataAccessException {
            for (byte[] each : this.keys) {
                if (this.now == null) {
                    // 如果没有基准时间，清空全部
                    connection.keyCommands().del(each);
                    continue;
                }
                // now 为时间戳取负数（即 score，如 -12345），与 Redis 存法 score=-created 一致
                if (this.desc) {
                    // 删除比 now 更早的消息。早的消息时间戳小，-timestamp 就大，删除 [now, 0] 区间
                    connection.zSetCommands().zRemRangeByScore(each, this.now, 0);
                } else {
                    // 删除比 now 更晚(新)的消息。晚的消息时间戳大，-timestamp 就小，删除 [-inf, now] 区间
                    connection.zSetCommands().zRemRangeByScore(each, Double.NEGATIVE_INFINITY, this.now);
                }
            }
            return null;
        }
    }

    @Getter
    @Builder
    public static class RedisPair {

        protected final List<byte[]> keys;

        protected final byte[] history;

        // 创建时间（用于排序）
        protected final Long created;

    }

    @ConditionalOnProperty(name = "history.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier("redis4array")
        protected RedisTemplate<String, Object> redis4array;

        @Value("${history.maxsize:200}")
        protected Integer maxsize;

        // SECONDS，默认30天
        @Value("${history.expire:2592000}")
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = HistoryStore.class)
        public HistoryStore historyStore() throws Exception {
            RedisHistoryStore historyStore = new RedisHistoryStore();
            BeanUtils.copyProperties(this, historyStore);
            log.info("RedisHistoryStore inited: maxsize={}, expire={}", historyStore.getMaxsize(), historyStore.getExpire());
            return historyStore;
        }
    }
}
