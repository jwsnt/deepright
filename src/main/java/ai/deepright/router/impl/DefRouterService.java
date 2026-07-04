package ai.deepright.router.impl;

import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.deepright.feature.FeatureFlag;
import ai.open.right.WorkflowException;
import ai.open.right.config.RedisConfig;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SpinExec;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.router.RouterAgent;
import ai.deepright.router.RouterDevice;
import ai.deepright.router.RouterService;
import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisOperations;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.SessionCallback;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
public class DefRouterService implements RouterService {

    public static final String NAME = "default_router_service";

    protected RedisTemplate<String, Object> redis4array;

    protected Cache<String, Boolean> cache4expire;

    protected ExecutorService executor;

    // 默认外部团队联系人
    protected List<String> defContact;

    protected Integer timeout;

    protected Integer circle;

    protected Integer expire;

    protected Integer update;

    // 仅用于阻止单实例并发更新
    protected Integer cache;

    @PostConstruct
    public void init() throws Exception {
        Assert.isTrue(this.update > this.cache, "The cli env config is invalid: " + this.update + " / " + this.cache);
        this.cache4expire = CacheBuilder.newBuilder().expireAfterWrite(this.cache, TimeUnit.MILLISECONDS).build();
    }

    @Override
    public Boolean hasHeartbeat(WorkflowTask workTask, String agent) throws Exception {
        return this.hasHeartbeat(new RouterDevice(workTask, agent));
    }

    @Override
    public Boolean hasHeartbeat(WorkflowTask workTask) throws Exception {
        return this.hasHeartbeat(new RouterDevice(workTask));
    }

    @Override
    public Boolean hasHeartbeat(RouterDevice routerDevice) throws Exception {
        // 检查Device+Key是否存在
        RouterDevice cacheDevice = this.fetch(routerDevice);
        Assert.notNull(cacheDevice, "The device [" + routerDevice.key() + "] is not exists");
        Assert.isTrue(!cacheDevice.isExpired(this.expire), "The device [" + routerDevice.key() + "] is expired");
        return true;
    }

    // 更新心跳
    @Override
    public void heartbeat(WorkflowTask workTask) throws Exception {
        RouterAgent[] agents = FeatureUtils.buildRouterAgents(workTask);
        if (log.isInfoEnabled()) {
            log.info("The device={} updated heartbeat={}", workTask.getDevice(), ArrayUtils.getLength(agents));
        }
        if (!ArrayUtils.isEmpty(agents)) {
            List<String> device = new ArrayList<String>();
            List<String> agent = new ArrayList<String>();
            List<byte[]> val = new ArrayList<byte[]>();
            for (RouterAgent each : agents) {
                RouterDevice routerDevice = each.buildRouterDevice(workTask);
                // 与Fetch/Get时取值对齐
                val.add(GzipUtils.compress(JsonUtils.write(routerDevice).getBytes(StandardCharsets.UTF_8)));
                device.add(this.getRouter(routerDevice.getDevice()));
                agent.add(routerDevice.getAgent());
            }
            // 先保存Redis，然后刷新Local Cache（设备维度，用于防止短时间重复提交）
            this.cache4expire.get(workTask.getDevice(), new RouterSetExec(this.redis4array, device, agent, val, this.expire, this.timeout, this.circle));
        }
    }

    @Override
    public RouterDevice fetch(WorkflowTask workTask, String device, String agent) throws Exception {
        return this.fetch(new RouterDevice(workTask, device, agent));
    }

    @Override
    public RouterDevice fetch(WorkflowTask workTask, String agent) throws Exception {
        return this.fetch(new RouterDevice(workTask, agent));
    }

    @Override
    public RouterDevice fetch(String router, String agent) throws Exception {
        // 检查Device+Key是否存在
        Object value = new RouterFetchExec(this.redis4array, this.timeout, this.circle, this.getRouter(router), agent).exec();
        return value != null ? JsonUtils.read(GzipUtils.decompress((byte[]) value), RouterDevice.class) : null;
    }

    @Override
    public RouterDevice fetch(RouterDevice routerDevice) throws Exception {
        // 检查Device+Key是否存在
        return this.fetch(routerDevice.getDevice(), routerDevice.getAgent());
    }

    @Override
    public RouterDevice fetch(WorkflowTask workTask) throws Exception {
        return this.fetch(new RouterDevice(workTask));
    }

    @Override
    public RouterDevice fetch(String key) throws Exception {
        String[] parts = SplitUtils.split(key);
        return this.fetch(parts[0], parts[1]);
    }

    @Override
    public List<RouterDevice> router(WorkflowTask workTask) throws Exception {
        List<RouterDevice> router = new ArrayList<RouterDevice>();
        // 未阻止加载Router且开启了Router
        // Task主动关闭团队，防止递归
        if (this.allowedRouter(workTask)) {
            List<String> access = this.getRouter(RouterDevice.contact(workTask, this.defContact));
            if (log.isInfoEnabled()) {
                log.info("The router access={}", access);
            }
            Object result = new RouterGetAllExec(this.redis4array, access, this.timeout, this.circle).exec();
            if (result != null) {
                List<List<Object>> cached = List.class.cast(result);
                // 用于存放过期
                List<RouterDevice> expired = null;
                if (!CollectionUtils.isEmpty(cached)) {
                    for (List<Object> each : cached) {
                        if (each != null) {
                            for (Object inner : each) {
                                try {
                                    RouterDevice routerDevice = JsonUtils.read(GzipUtils.decompress((byte[]) inner), RouterDevice.class);
                                    Boolean isExpire = routerDevice.isExpired(this.expire);
                                    if (!isExpire && routerDevice.getEnabled()) {
                                        // 去掉本机当前Agent
                                        if (!StringUtils.equalsIgnoreCase(routerDevice.key(), RouterDevice.key(workTask))) {
                                            // 去除敏感信息
                                            routerDevice.maskWorkspace().setMetadata(null);
                                            router.add(routerDevice);
                                        }
                                    } else if (isExpire) {
                                        // 过期的，最后异步删除
                                        expired = expired != null ? expired : new ArrayList<RouterDevice>();
                                        expired.add(routerDevice);
                                    }
                                } catch (Exception e) {
                                    log.error(e.getMessage(), e);
                                }
                            }
                        }
                    }
                    this.expire(expired);
                }
            }
        }
        return router;
    }

    protected void expire(List<RouterDevice> routerDevice) throws Exception {
        if (!CollectionUtils.isEmpty(routerDevice)) {
            List<String> device = new ArrayList<String>();
            List<String> agent = new ArrayList<String>();
            for (RouterDevice each : routerDevice) {
                // 与Set时取值对齐
                device.add(this.getRouter(each.getDevice()));
                agent.add(each.getAgent());
            }
            this.executor.execute(new RouterDelExec(this.redis4array, device, agent, this.timeout, this.circle));
        }
    }

    protected List<String> getRouter(List<String> access) throws Exception {
        List<String> router = new ArrayList<>();
        for (String each : access) {
            router.add(this.getRouter(each));
        }
        return router;
    }

    protected String getRouter(String router) throws Exception {
        return RedisConfig.DOMAIN + DefRouterService.class.getSimpleName() + "_r_" + router;
    }

    protected Boolean allowedRouter(WorkflowTask workTask) throws Exception {
        // 开启Router
        return !RouterDevice.disable(workTask);
    }

    public static class RouterSetRedisCallable implements SessionCallback<Object> {

        protected final List<String> router;

        protected final List<String> agent;

        protected final List<byte[]> val;

        protected final Integer expire;

        public RouterSetRedisCallable(List<String> router, List<String> agent, List<byte[]> val, Integer expire) {
            this.expire = expire;
            this.router = router;
            this.agent = agent;
            this.val = val;
        }

        @Override
        @SuppressWarnings("unchecked")
        public Object execute(RedisOperations operations) {
            // 自身
            for (int index = 0; index < this.router.size(); index++) {
                String r = this.router.get(index);
                String n = this.agent.get(index);
                byte[] v = this.val.get(index);
                // 设备集
                operations.opsForHash().put(r, n, v);
                operations.expire(r, this.expire, TimeUnit.MILLISECONDS);
            }
            return null;
        }
    }

    public static class RouterDelRedisCallable implements SessionCallback<Object> {

        protected final List<String> router;

        protected final List<String> agent;

        public RouterDelRedisCallable(List<String> router, List<String> agent) {
            this.router = router;
            this.agent = agent;
        }

        @Override
        @SuppressWarnings("unchecked")
        public Object execute(RedisOperations operations) {
            for (int index = 0; index < this.router.size(); index++) {
                operations.opsForHash().delete(this.router.get(index), this.agent.get(index));
            }
            return null;
        }
    }

    public static class RouterFetchExec extends SpinExec {

        protected final RedisTemplate<String, Object> redis4array;

        protected final String router;

        protected final String agent;

        public RouterFetchExec(RedisTemplate<String, Object> redis4array, Integer timeout, Integer circle, String router, String agent) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.router = router;
            this.agent = agent;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                return this.redis4array.opsForHash().get(this.router, this.agent);
            } catch (Exception e) {
                log.error(e.getMessage(), e);
                return null;
            }
        }
    }

    public static class RouterGetAllExec extends SpinExec implements SessionCallback<Object> {

        protected final RedisTemplate<String, Object> redis4array;

        protected final List<String> access;

        public RouterGetAllExec(RedisTemplate<String, Object> redis4array, List<String> access, Integer timeout, Integer circle) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.access = access;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                // 取出Router下的所有设备Key
                return this.redis4array.executePipelined(this);
            } catch (Exception e) {
                log.error(e.getMessage(), e);
                return null;
            }
        }

        @Override
        @SuppressWarnings("unchecked")
        public Object execute(RedisOperations operations) {
            for (String key : this.access) {
                operations.opsForHash().values(key);
            }
            return null;
        }
    }

    public static class RouterSetExec extends SpinExec implements Callable<Boolean> {

        protected final RedisTemplate<String, Object> redis4array;

        protected final List<String> device;

        protected final List<String> agent;

        protected final List<byte[]> val;

        protected final Integer timeout;

        protected final Integer expire;

        protected final Integer circle;

        public RouterSetExec(RedisTemplate<String, Object> redis4array, List<String> device, List<String> agent, List<byte[]> val, Integer expire, Integer timeout, Integer circle) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.timeout = timeout;
            this.circle = circle;
            this.expire = expire;
            this.device = device;
            this.agent = agent;
            this.val = val;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                Object result = this.redis4array.executePipelined(new RouterSetRedisCallable(this.device, this.agent, this.val, this.expire));
                if (log.isInfoEnabled()) {
                    log.info("The router was registered, device={}, agent={}, result={}", this.device, this.agent, result);
                }
                return result;
            } catch (Exception e) {
                log.error(e.getMessage(), e);
                return null;
            }
        }

        @Override
        public Boolean call() throws Exception {
            this.doExec();
            return true;
        }
    }

    public static class RouterDelExec extends SpinExec implements Runnable {

        protected final RedisTemplate<String, Object> redis4array;

        protected final List<String> router;

        protected final List<String> agent;

        public RouterDelExec(RedisTemplate<String, Object> redis4array, List<String> router, List<String> agent, Integer timeout, Integer circle) {
            super(timeout, circle);
            this.redis4array = redis4array;
            this.router = router;
            this.agent = agent;
        }

        @Override
        public Object doExec() throws Exception {
            try {
                Object result = this.redis4array.executePipelined(new RouterDelRedisCallable(this.router, this.agent));
                if (log.isInfoEnabled()) {
                    log.info("The router was destroyed, router={}, agent={}, result={}", this.router, this.agent, result);
                }
                return result;
            } catch (Exception e) {
                log.error(e.getMessage(), e);
                return null;
            }
        }

        @Override
        public void run() {
            try {
                this.doExec();
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Autowired
        protected RedisTemplate<String, Object> redis4array;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executor;

        // Timeout内尝试Circle次
        @Value("${router.timeout:10000}")
        protected Integer timeout;

        @Value("${router.circle:10}")
        protected Integer circle;

        // 需要大于CLI/Get最大可能时间
        @Value("${router.expire:360000}")
        protected Integer expire;

        @Value("${router.update:15000}")
        protected Integer update;

        @Value("${router.cache:1500}")
        protected Integer cache;

        @Value("${router.contact:}")
        protected String contact;

        @Bean(DefRouterService.NAME)
        @ConditionalOnMissingBean(name = DefRouterService.NAME)
        public DefRouterService defaultRouterService() throws Exception {
            DefRouterService defaultRouterService = new DefRouterService();
            BeanUtils.copyProperties(this, defaultRouterService);
            if (!StringUtils.isEmpty(this.contact)) {
                defaultRouterService.setDefContact(Arrays.asList(StringUtils.split(this.contact, ",")));
            }
            log.info("DefaultRouterService inited");
            return defaultRouterService;
        }
    }

}
