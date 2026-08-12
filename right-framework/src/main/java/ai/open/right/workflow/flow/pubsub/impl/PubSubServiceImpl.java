package ai.open.right.workflow.flow.pubsub.impl;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.SpinExec;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.pubsub.PubSubConfig;
import ai.open.right.workflow.flow.pubsub.PubSubFormatter;
import ai.open.right.workflow.flow.pubsub.PubSubQuery;
import ai.open.right.workflow.flow.pubsub.PubSubService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
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
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Slf4j
@Setter
@Getter
public class PubSubServiceImpl implements PubSubService {

    public static final String KEY = "__pub__sub";

    protected RedisTemplate<String, Object> redis4event;

    protected Map<String, PubSubFormatter> formatters;

    protected NotifierService notifierService;

    // 自旋间隔
    protected Integer interval;

    // 等待Sub反馈的超时
    protected Integer timeout;

    protected Integer circle;

    // SECONDS
    // Pub持久化时间
    protected Integer expire;

    @Override
    // 消费
    public String sub(PubSubConfig pubSubConfig, WorkflowTask workTask) throws Exception {
        Assert.notNull(this.redis4event, "Redis4event can not be empty");
        PubSubQuery pubSubQuery = this.pubSubQuery(workTask);
        // 生成Sub Key并绑定
        String key = this.key();
        pubSubQuery.setKey(key);
        try {
            if (log.isDebugEnabled()) {
                log.debug("Sub key and query: key={},query={}", key, pubSubQuery);
            }
            Segment segment = this.segment(pubSubConfig, workTask, pubSubQuery.getQuery(), key);
            this.notifierService.notify(this.format(pubSubConfig, pubSubQuery, workTask, segment), workTask, workTask);
            return this.sub(pubSubConfig, key);
        } catch (Exception e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            // 失败时，如果有默认Answer则返回
            if (pubSubQuery.hasAnswer()) {
                String answer = pubSubQuery.getAnswer();
                if (log.isInfoEnabled()) {
                    log.info("Sub key and default answer: key={},answer={}", key, answer);
                }
                this.storeDefault(pubSubConfig, pubSubQuery, workTask, answer);
                return answer;
            }
            // 失败时，如果有配置Answer则返回
            if (pubSubConfig != null && pubSubConfig.hasReply()) {
                String answer = pubSubConfig.getReply();
                if (log.isInfoEnabled()) {
                    log.info("Sub key and config answer: key={},answer={}", key, answer);
                }
                this.storeReply(pubSubConfig, pubSubQuery, workTask, answer);
                return answer;
            }
            throw e;
        }
    }

    @Override
    public String sub(PubSubConfig pubSubConfig, String key) throws Exception {
        Integer timeout = pubSubConfig != null ? pubSubConfig.getTimeout4Llm(this.timeout) : this.timeout;
        if (log.isDebugEnabled()) {
            log.debug("Sub timeout={}", timeout);
        }
        return this.sub(timeout, key);
    }

    @Override
    public String sub(Integer timeout, String key) throws Exception {
        Assert.notNull(this.redis4event, "Redis4event can not be empty");
        // 如果为大量Pub/Sub，覆盖为自旋+超时，避免长期占用Redis连接
        List<List<Object>> result = (List<List<Object>>) (List<?>) new SubExec(this.redis4event, this.interval, this.timeout, this.circle, key).exec();
        if (log.isDebugEnabled()) {
            log.debug("Sub key and redis result: key={},result={}", key, result);
        }
        Assert.notEmpty(result, "Sub result can not be empty");
        List<Object> data = result.getFirst();
        Assert.notEmpty(data, "Sub data can not be empty");
        String answer = new String((byte[]) data.get(1), StandardCharsets.UTF_8);
        if (log.isInfoEnabled()) {
            log.info("Sub key and answer: key={},answer={}", key, answer);
        }
        return answer;
    }

    @Override
    public String sub(String key) throws Exception {
        return this.sub(this.timeout, key);
    }

    @Override
    // 发布
    public void pub(PubSubConfig pubSubConfig, WorkflowTask workTask) throws Exception {
        String key = workTask.getMetadata(PubSubServiceImpl.KEY, String.class);
        String val = workTask.getQuery();
        this.pub(pubSubConfig, key, val);
    }

    @Override
    public void pub(PubSubConfig pubSubConfig, String k, String v) throws Exception {
        this.pub(this.expire, k, v);
    }

    @Override
    public void pub(Integer expire, String k, String v) throws Exception {
        Assert.notNull(this.redis4event, "Redis4event can not be empty");
        Assert.hasText(v, "Pub value can not be empyt: " + v);
        if (log.isDebugEnabled()) {
            log.debug("Pub key={}", k);
        }
        List<Object> result = this.redis4event.executePipelined(new RedisPubCallback(k, v, expire));
        if (log.isInfoEnabled()) {
            log.info("Pub key and redis result: key={},answer={}", k, result);
        }
    }

    @Override
    public void pub(String k, String v) throws Exception {
        this.pub(this.expire, k, v);
    }

    // 默认不处理，用于子类覆盖
    protected void storeDefault(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, String answer) throws Exception {
    }

    // 默认不处理，用于子类覆盖
    // 用于子类覆盖
    protected void storeReply(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, String answer) throws Exception {
    }

    // 格式化Segment
    protected Segment format(PubSubConfig pubSubConfig, PubSubQuery pubSubQuery, WorkflowTask workTask, Segment segment) throws Exception {
        if (CollectionUtils.isEmpty(this.formatters) || pubSubConfig == null || pubSubConfig.hasFormatter()) {
            return segment;
        }
        PubSubFormatter pubSubFormatter = this.formatters.get(pubSubConfig.getFormatter());
        if (log.isDebugEnabled()) {
            log.debug("Pub formatter: key{}:{}", pubSubConfig.getFormatter(), pubSubFormatter);
        }
        Assert.notNull(pubSubFormatter, "PubSub formatter can not be empty: " + pubSubConfig.getFormatter());
        return pubSubFormatter.format(pubSubConfig, pubSubQuery, workTask, segment);
    }

    // 构建Pub Segment
    protected Segment segment(PubSubConfig pubSubConfig, WorkflowTask workTask, String query, String key) throws Exception {
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                // 指定通知方式
                .notifier(pubSubConfig != null ? pubSubConfig.getNotifier(Notifier.SOURCE) : Notifier.SOURCE)
                // 指定状态码
                .code(pubSubConfig != null ? pubSubConfig.getCode() : ProtocolCode.C200)
                .metadata(Collections.singletonMap(PubSubServiceImpl.KEY, key))
                .content(query != null ? new StringBuffer(query) : null)
                .build();
        return Segment.build(workTask, segmentConfig);
    }

    protected PubSubQuery pubSubQuery(WorkflowTask workTask) throws Exception {
        Assert.hasText(workTask.getQuery(), "PubSub workTask query can not be empty");
        if (log.isDebugEnabled()) {
            log.debug("PubSubQuery content={}", workTask.getQuery());
        }
        PubSubQuery pubSubQuery = workTask.getObjectQuery(PubSubQuery.class);
        Assert.hasText(pubSubQuery.getQuery(), "PubSub query can not be empty");
        return pubSubQuery;
    }

    protected String key() throws Exception {
        return UUID.randomUUID().toString();
    }

    public static class RedisPubCallback implements RedisCallback<Void> {

        protected final Integer timeout;

        protected final String value;

        protected final String key;

        public RedisPubCallback(String key, String value, Integer timeout) {
            this.timeout = timeout;
            this.value = value;
            this.key = key;
        }

        @Override
        public Void doInRedis(RedisConnection connection) throws DataAccessException {
            byte[] kBytes = this.key.getBytes(StandardCharsets.UTF_8);
            connection.listCommands().rPush(kBytes, this.value.getBytes(StandardCharsets.UTF_8));
            connection.keyCommands().expire(kBytes, this.timeout);
            return null;
        }
    }

    public static class RedisSubCallback implements RedisCallback<List<byte[]>> {

        protected final Integer timeout;

        protected final String key;

        public RedisSubCallback(String key, Integer timeout) {
            this.timeout = timeout;
            this.key = key;
        }

        @Override
        public List<byte[]> doInRedis(RedisConnection connection) throws DataAccessException {
            byte[] key = this.key.getBytes(StandardCharsets.UTF_8);
            return connection.listCommands().bLPop(this.timeout, key);
        }
    }

    public static class SubExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4event;

        protected final Integer interval;

        protected final String key;

        public SubExec(RedisTemplate<String, Object> redis4event, Integer interval, Integer seconds, Integer cycles, String key) {
            super(seconds, cycles);
            this.redis4event = redis4event;
            this.interval = interval;
            this.key = key;
        }

        @Override
        public Object doExec() throws Exception {
            List<List<Object>> result = (List<List<Object>>) (List<?>) this.redis4event.executePipelined(new RedisSubCallback(this.key, this.interval));
            if (log.isDebugEnabled()) {
                log.debug("Sub key and redis result: key={},result={}", this.key, result);
            }
            return result;
        }
    }

    @ConditionalOnProperty(name = "pubsub.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected RedisTemplate<String, Object> redis4event;

        @Autowired(required = false)
        protected Map<String, PubSubFormatter> formatters;

        @Autowired
        protected NotifierService notifierService;

        // 自旋间隔
        @Value("${pubsub.interval:100}")
        protected Integer interval;

        @Value("${pubsub.timeout:1800000}")
        // 等待Sub反馈的超时
        protected Integer timeout;

        // 自旋次数
        @Value("${pubsub.circle:50}")
        protected Integer circle;

        // SECONDS
        @Value("${pubsub.expire:3000}")
        // Pub持久化时间
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = PubSubService.class)
        public PubSubService pubSubService() throws Exception {
            PubSubServiceImpl pubSubService = new PubSubServiceImpl();
            BeanUtils.copyProperties(this, pubSubService);
            log.info("PubSubServiceImpl inited: interval={}, circle={}, timeout={}, expire={}", pubSubService.getInterval(), pubSubService.getCircle(), pubSubService.getTimeout(), pubSubService.getExpire());
            return pubSubService;
        }
    }
}
