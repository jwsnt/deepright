package ai.open.right.config;

import org.junit.Assert;
import org.junit.Test;
import org.springframework.boot.autoconfigure.condition.AnyNestedCondition;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.ConfigurationCondition;

/**
 * {@link RedisConfig.RedisConfigCondition} 为 {@link AnyNestedCondition}：任一嵌套
 * {@link ConditionalOnProperty} 满足时整体匹配。当前工程 Spring 组件版本不一致，无法在单测中启动
 * {@code ApplicationContext} 做端到端校验，此处覆盖类型结构与注解契约。
 */
public class RedisConfigConditionTest {

    @Test
    public void conditionType_extendsAnyNestedCondition() {
        Assert.assertTrue(AnyNestedCondition.class.isAssignableFrom(RedisConfig.RedisConfigCondition.class));
    }

    @Test
    public void constructor_usesRegisterBeanPhase() {
        RedisConfig.RedisConfigCondition condition = new RedisConfig.RedisConfigCondition();
        Assert.assertEquals(
                ConfigurationCondition.ConfigurationPhase.REGISTER_BEAN,
                condition.getConfigurationPhase());
    }

    @Test
    public void nestedClasses_conditionalOnExpectedProperties() {
        assertConditionalOnProperty(RedisConfig.RedisConfigCondition.OnListenerEnable.class, "event.listener.redis.enable");
        assertConditionalOnProperty(RedisConfig.RedisConfigCondition.OnHistoryEnable.class, "history.enable");
        assertConditionalOnProperty(RedisConfig.RedisConfigCondition.OnCommandEnable.class, "command.enable");
        assertConditionalOnProperty(RedisConfig.RedisConfigCondition.OnTokenEnable.class, "token.enable");
        assertConditionalOnProperty(RedisConfig.RedisConfigCondition.OnBlockEnable.class, "block.enable");
    }

    private static void assertConditionalOnProperty(Class<?> nested, String expectedName) {
        ConditionalOnProperty ann = nested.getAnnotation(ConditionalOnProperty.class);
        Assert.assertNotNull(nested.getSimpleName() + " 应有 @ConditionalOnProperty", ann);
        Assert.assertArrayEquals(new String[] { expectedName }, ann.name());
        Assert.assertEquals("true", ann.havingValue());
    }
}
