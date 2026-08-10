package ai.open.right.workflow.flow.llm.store.history.impl;

import org.junit.Assert;
import org.junit.Test;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;


public class RedisPairTest {

    // 测试正常场景：完整参数构建对象
    @Test
    public void testBuilderWithFullParams() {
        // 准备测试数据
        byte[] key1 = "key1".getBytes();
        byte[] key2 = "key2".getBytes();
        List<byte[]> keys = Arrays.asList(key1, key2);
        byte[] history = "test_history".getBytes();

        // 构建对象
        RedisHistoryStore.RedisPair redisPair = RedisHistoryStore.RedisPair.builder()
                .keys(keys)
                .history(history)
                .build();

        // 验证 getter 方法
        Assert.assertNotNull("keys 不应为 null", redisPair.getKeys());
        Assert.assertEquals("keys 数量不匹配", 2, redisPair.getKeys().size());
        Assert.assertSame("第一个 key 引用不匹配", key1, redisPair.getKeys().get(0));
        Assert.assertSame("第二个 key 引用不匹配", key2, redisPair.getKeys().get(1));
        Assert.assertSame("history 引用不匹配", history, redisPair.getHistory());
    }

    // 测试边界场景：keys 为 null
    @Test
    public void testBuilderWithNullKeys() {
        byte[] history = "history".getBytes();

        RedisHistoryStore.RedisPair redisPair = RedisHistoryStore.RedisPair.builder()
                .keys(null)
                .history(history)
                .build();

        Assert.assertNull("keys 应为 null", redisPair.getKeys());
        Assert.assertSame(history, redisPair.getHistory());
    }

    // 测试边界场景：history 为 null
    @Test
    public void testBuilderWithNullHistory() {
        List<byte[]> keys = Collections.singletonList("test_key".getBytes());

        RedisHistoryStore.RedisPair redisPair = RedisHistoryStore.RedisPair.builder()
                .keys(keys)
                .history(null)
                .build();

        Assert.assertEquals(keys, redisPair.getKeys());
        Assert.assertNull("history 应为 null", redisPair.getHistory());
    }

    // 测试边界场景：空 keys 列表
    @Test
    public void testBuilderWithEmptyKeys() {
        List<byte[]> emptyKeys = Collections.emptyList();
        byte[] history = new byte[0]; // 空字节数组

        RedisHistoryStore.RedisPair redisPair = RedisHistoryStore.RedisPair.builder()
                .keys(emptyKeys)
                .history(history)
                .build();

        Assert.assertTrue("keys 应为空列表", redisPair.getKeys().isEmpty());
        Assert.assertEquals("history 应为空字节数组", 0, redisPair.getHistory().length);
    }

    // 测试 getter 方法的返回值是否与构建时一致（引用一致性）
    @Test
    public void testGetterReturnValues() {
        byte[] key = "unique_key".getBytes();
        List<byte[]> keys = Collections.singletonList(key);
        byte[] history = "unique_history".getBytes();

        RedisHistoryStore.RedisPair redisPair = RedisHistoryStore.RedisPair.builder()
                .keys(keys)
                .history(history)
                .build();

        // 验证列表引用（若原始列表修改，getter 返回的列表也会变化，因为 List 是引用类型）
        Assert.assertEquals("keys 列表应包含新增元素", 1, redisPair.getKeys().size());

        // 验证字节数组引用（原始数组修改，getter 返回的数组也会变化）
        history[0] = 'X';
        Assert.assertEquals("history 数组应被修改", 'X', redisPair.getHistory()[0]);
    }

    // 测试不同对象的字段独立性
    @Test
    public void testIndependentInstances() {
        List<byte[]> keys1 = Arrays.asList("keyA".getBytes());
        byte[] history1 = "historyA".getBytes();
        
        List<byte[]> keys2 = Arrays.asList("keyB".getBytes());
        byte[] history2 = "historyB".getBytes();

        RedisHistoryStore.RedisPair pair1 = RedisHistoryStore.RedisPair.builder().keys(keys1).history(history1).build();
        RedisHistoryStore.RedisPair pair2 = RedisHistoryStore.RedisPair.builder().keys(keys2).history(history2).build();

        Assert.assertNotEquals("两个对象的 keys 应不同", pair1.getKeys(), pair2.getKeys());
        Assert.assertNotEquals("两个对象的 history 应不同", pair1.getHistory(), pair2.getHistory());
    }
}