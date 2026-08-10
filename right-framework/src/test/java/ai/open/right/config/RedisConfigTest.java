package ai.open.right.config;

import ai.open.right.release.ResourceReleaser;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.connection.RedisStandaloneConfiguration;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.RedisSerializer;
import org.springframework.data.redis.serializer.StringRedisSerializer;

public class RedisConfigTest {

    /** 使用真实工厂，避免 JDK 高版本下对 LettuceConnectionFactory 的 mock 限制 */
    private static LettuceConnectionFactory testLettuceFactory() {
        return new LettuceConnectionFactory(new RedisStandaloneConfiguration("127.0.0.1", 6379));
    }

    /** RedisConfig 仅有默认无参构造；@Value 未注入时 extreme 为 null */
    @Test
    public void testDefaultConstructor() {
        RedisConfig config = new RedisConfig();
        Assert.assertNull(config.getExtreme());
        Assert.assertNull(config.getRedis4funCall());
        Assert.assertNull(config.getRedis4event());
        Assert.assertNull(config.getRedis4array());
        Assert.assertNull(config.getRedis4chat());
    }

    @Test
    public void testRedis4funCall() throws Exception {
        RedisConfig config = new RedisConfig();
        config.setExtreme(false);
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> template = config.redis4funCall(lettuce);
        Assert.assertEquals(template.getConnectionFactory(), lettuce);
        Assert.assertEquals(template.getKeySerializer().getClass(), StringRedisSerializer.class);
        Assert.assertEquals(template.getValueSerializer().getClass(), RedisSerializer.byteArray().getClass());
    }

    /** extreme=true 时 !extreme 为 false，redis4funCall 复用 redis4array */
    @Test
    public void testRedis4funCall_extremeTrue_reusesRedis4array() throws Exception {
        RedisConfig config = new RedisConfig();
        config.setExtreme(true);
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> array = config.redis4array(lettuce);
        RedisTemplate<String, Object> funCall = config.redis4funCall(lettuce);
        Assert.assertSame(array, funCall);
    }

    @Test
    public void testRedis4array() throws Exception {
        RedisConfig config = new RedisConfig();
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> template = config.redis4array(lettuce);
        Assert.assertEquals(template.getConnectionFactory(), lettuce);
        Assert.assertEquals(template.getKeySerializer().getClass(), StringRedisSerializer.class);
        Assert.assertEquals(template.getValueSerializer().getClass(), RedisSerializer.byteArray().getClass());
    }

    /** extreme=true 时 !extreme 为 false，redis4event 复用 redis4array */
    @Test
    public void testRedis4event_extremeTrue_reusesRedis4array() throws Exception {
        RedisConfig config = new RedisConfig();
        config.setExtreme(true);
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> array = config.redis4array(lettuce);
        RedisTemplate<String, Object> event = config.redis4event(lettuce);
        Assert.assertSame(array, event);
    }

    @Test
    public void testRedis4event() throws Exception {
        RedisConfig config = new RedisConfig();
        config.setExtreme(false);
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> template = config.redis4event(lettuce);
        Assert.assertEquals(template.getConnectionFactory(), lettuce);
        Assert.assertEquals(template.getKeySerializer().getClass(), StringRedisSerializer.class);
        Assert.assertEquals(template.getValueSerializer().getClass(), RedisSerializer.byteArray().getClass());
    }

    /** extreme=true 时 !extreme 为 false，redis4chat 复用 redis4array */
    @Test
    public void testRedis4chat_extremeTrue_reusesRedis4array() throws Exception {
        RedisConfig config = new RedisConfig();
        config.setExtreme(true);
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> array = config.redis4array(lettuce);
        RedisTemplate<String, Object> chat = config.redis4chat(lettuce);
        Assert.assertSame(array, chat);
    }

    @Test
    public void testRedis4chat() throws Exception {
        RedisConfig config = new RedisConfig();
        config.setExtreme(false);
        LettuceConnectionFactory lettuce = testLettuceFactory();
        RedisTemplate<String, Object> template = config.redis4chat(lettuce);
        Assert.assertEquals(template.getConnectionFactory(), lettuce);
        Assert.assertEquals(template.getKeySerializer().getClass(), StringRedisSerializer.class);
        Assert.assertEquals(template.getValueSerializer().getClass(), RedisSerializer.byteArray().getClass());
    }

    @Test
    public void testSetGet() {
        ResourceReleaser resourceConfig = new ResourceReleaser();
        resourceConfig.setFile("FILE");
        Assert.assertEquals("FILE", resourceConfig.getFile());
        ResourceReleaser.InitConfig initConfig = new ResourceReleaser.InitConfig();
        initConfig.setFile("FILE");
        initConfig.setSuffix("SUFFIX");
        initConfig.setRelease(true);
        Assert.assertEquals("FILE", initConfig.getFile());
        Assert.assertEquals("SUFFIX", initConfig.getSuffix());
        Assert.assertTrue(initConfig.getRelease());
    }

    @org.junit.jupiter.api.Test
    public void testRedisConfig() {
        RedisConfig config = new RedisConfig();
        org.junit.jupiter.api.Assertions.assertNotNull(config);
    }

    @org.junit.jupiter.api.Test
    public void testRedis4funCallNull() {
        RedisConfig config = new RedisConfig();
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            config.redis4funCall(null);
        });
    }

}
