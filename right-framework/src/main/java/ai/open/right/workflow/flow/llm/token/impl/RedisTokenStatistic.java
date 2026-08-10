package ai.open.right.workflow.flow.llm.token.impl;

import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import jakarta.annotation.PostConstruct;
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
import org.springframework.util.CollectionUtils;

import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

@Slf4j
@Getter
@Setter
public class RedisTokenStatistic implements TokenStatistic {

    public static final TokenData EMPTY = TokenData.builder().build();

    public static final String SUFFIX_THINKING = "_i";

    public static final String SUFFIX_INPUT = "_p";

    public static final String SUFFIX_TOKEN = "_t";

    public static final String SUFFIX_CACHE = "_c";

    protected RedisTemplate<String, Object> redis4array;

    protected Set<String> models;

    // SECONDS
    // 统计缓存时间
    protected Integer expire;

    @PostConstruct
    public void init() throws Exception {
        this.models = ConcurrentHashMap.newKeySet();
    }

    @Override
    public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
        try {
            if (log.isDebugEnabled()) {
                log.debug("Redis token statistic data={}", tokenData);
            }
            Assert.notNull(this.redis4array, "Redis4array can not be empty");
            String key = this.getKey(providerRequest.getMessage().getBiz(), providerRequest.getMessage().getChat(), providerRequest.getMessage().getDevice(), this.addModel(providerRequest, this.getModel(providerRequest)));
            String key4thinking = key + RedisTokenStatistic.SUFFIX_THINKING;
            String key4input = key + RedisTokenStatistic.SUFFIX_INPUT;
            String key4token = key + RedisTokenStatistic.SUFFIX_TOKEN;
            String key4cache = key + RedisTokenStatistic.SUFFIX_CACHE;
            this.stat(key4thinking, key4input, key4token, key4cache, tokenData);
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    @Override
    public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
        Assert.notNull(this.redis4array, "Redis4array can not be empty");
        List<String> key = this.getKey(dimension.getBiz(), dimension.getChat(), dimension.getDevice(), model);
        List<String> all = new ArrayList<String>(key.size() * 2);
        for (String each : key) {
            all.add(each + RedisTokenStatistic.SUFFIX_TOKEN);
            all.add(each + RedisTokenStatistic.SUFFIX_CACHE);
        }
        List<byte[]> values = (List<byte[]>) (List<?>) this.redis4array.opsForValue().multiGet(all);
        if (CollectionUtils.isEmpty(values)) {
            return Collections.emptyList();
        }
        List<TokenData> tokenData = new ArrayList<>();
        for (int i = 0; i < values.size(); i += 2) {
            byte[] tBytes = values.get(i);
            byte[] cBytes = values.get(i + 1);
            tokenData.add(TokenData.builder()
                    .total(tBytes != null ? Integer.parseInt(new String(tBytes, StandardCharsets.UTF_8)) : 0)
                    .cache(cBytes != null ? Integer.parseInt(new String(cBytes, StandardCharsets.UTF_8)) : 0)
                    .build());
        }
        return tokenData;
    }

    @Override
    public List<TokenData> readAll(Dimension dimension) throws Exception {
        return this.readAll(dimension, new ArrayList<String>(this.models));
    }

    @Override
    public TokenData read(Dimension dimension, String model) throws Exception {
        Assert.notNull(this.redis4array, "Redis4array can not be empty");
        String key = this.getKey(dimension.getBiz(), dimension.getChat(), dimension.getDevice(), model);
        String key4token = key + RedisTokenStatistic.SUFFIX_TOKEN;
        String key4cache = key + RedisTokenStatistic.SUFFIX_CACHE;
        List<byte[]> values = (List<byte[]>) (List<?>) this.redis4array.opsForValue().multiGet(Arrays.asList(key4token, key4cache));
        if (CollectionUtils.isEmpty(values)) {
            return RedisTokenStatistic.EMPTY;
        }
        byte[] tBytes = values.getFirst();
        byte[] cBytes = values.getLast();
        return TokenData.builder()
                .total(tBytes != null ? Integer.parseInt(new String(tBytes, StandardCharsets.UTF_8)) : 0)
                .cache(cBytes != null ? Integer.parseInt(new String(cBytes, StandardCharsets.UTF_8)) : 0)
                .build();
    }

    @Override
    public TokenData read(Dimension dimension) throws Exception {
        return this.read(dimension, "");
    }

    @Override
    public Set<String> models() throws Exception {
        return Collections.unmodifiableSet(this.models);
    }

    protected void stat(String key4thinking, String key4input, String key4token, String key4cache, TokenData tokenData) throws Exception {
        Assert.notNull(this.redis4array, "Redis4array can not be empty");
        if (tokenData.hasData()) {
            List<Object> result = this.redis4array.executePipelined(new RedisTokenStatisticCallback(key4thinking.getBytes(StandardCharsets.UTF_8), key4input.getBytes(StandardCharsets.UTF_8), key4token.getBytes(StandardCharsets.UTF_8), key4cache.getBytes(StandardCharsets.UTF_8), tokenData, this.expire));
            if (log.isInfoEnabled()) {
                log.info("Redis token statistic key4token={}, key4cache={}, tokenData={}, result={}", key4token, key4cache, tokenData, result);
            }
        }
    }

    protected List<String> getKey(String biz, String chat, String device, List<String> model) throws Exception {
        List<String> keys = new ArrayList<String>(model.size());
        for (String each : model) {
            keys.add(this.getKey(biz, chat, device, each));
        }
        return keys;
    }

    protected String getKey(String biz, String chat, String device, String model) throws Exception {
        return RedisConfig.DOMAIN + RedisTokenStatistic.class.getSimpleName() + device + model;
    }

    protected String addModel(ProviderRequest providerRequest, String model) throws Exception {
        if (!StringUtils.isEmpty(model)) {
            this.models.add(model);
        }
        return model;
    }

    // 默认为空，子类复写
    protected String getModel(ProviderRequest providerRequest) throws Exception {
        return "";
    }

    @ConditionalOnProperty(name = "token.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected RedisTemplate<String, Object> redis4array;

        // SECONDS，默认30天
        @Value("${token.statistics.expire:2592000}")
        // 统计缓存时间
        protected Integer expire;

        @Bean
        @ConditionalOnMissingBean(value = TokenStatistic.class)
        public TokenStatistic tokenStatistic() throws Exception {
            RedisTokenStatistic redisTokenStatistic = new RedisTokenStatistic();
            BeanUtils.copyProperties(this, redisTokenStatistic);
            log.info("RedisTokenStatistic inited");
            return redisTokenStatistic;
        }
    }
}
