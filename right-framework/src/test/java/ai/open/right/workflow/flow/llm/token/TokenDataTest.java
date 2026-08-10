package ai.open.right.workflow.flow.llm.token;

import org.junit.Assert;
import org.junit.Test;

/**
 * TokenData 单测：覆盖所有属性（thinking, input, token, cache）、NoArgsConstructor、AllArgsConstructor、Builder 及 getter/hasData 行为
 */
public class TokenDataTest {

    /** NoArgsConstructor：无参构造后各 getter 返回 0，hasData 为 false */
    @Test
    public void testNoArgsConstructor() {
        TokenData data = new TokenData();
        Assert.assertEquals(Integer.valueOf(0), data.getThinking());
        Assert.assertEquals(Integer.valueOf(0), data.getInput());
        Assert.assertEquals(Integer.valueOf(0), data.getTotal());
        Assert.assertEquals(Integer.valueOf(0), data.getCache());
        Assert.assertFalse(data.hasData());
    }

    /** AllArgsConstructor(thinking, input, token, cache)：全参构造后各 getter 返回对应值 */
    @Test
    public void testAllArgsConstructor() {
        TokenData data = new TokenData(1, 2, 10, 20);
        Assert.assertEquals(Integer.valueOf(1), data.getThinking());
        Assert.assertEquals(Integer.valueOf(2), data.getInput());
        Assert.assertEquals(Integer.valueOf(10), data.getTotal());
        Assert.assertEquals(Integer.valueOf(20), data.getCache());
        Assert.assertTrue(data.hasData());
    }

    /** AllArgsConstructor 全 null：各 getter 返回 0，hasData 为 false */
    @Test
    public void testAllArgsConstructor_withNulls() {
        TokenData data = new TokenData(null, null, null, null);
        Assert.assertEquals(Integer.valueOf(0), data.getThinking());
        Assert.assertEquals(Integer.valueOf(0), data.getInput());
        Assert.assertEquals(Integer.valueOf(0), data.getTotal());
        Assert.assertEquals(Integer.valueOf(0), data.getCache());
        Assert.assertFalse(data.hasData());
    }

    /** AllArgsConstructor 部分 null：未传的为 null，对应 getter 返回 0 */
    @Test
    public void testAllArgsConstructor_partialNull() {
        TokenData data = new TokenData(5, null, 10, null);
        Assert.assertEquals(Integer.valueOf(5), data.getThinking());
        Assert.assertEquals(Integer.valueOf(0), data.getInput());
        Assert.assertEquals(Integer.valueOf(10), data.getTotal());
        Assert.assertEquals(Integer.valueOf(0), data.getCache());
        Assert.assertTrue(data.hasData());
    }

    /** getThinking：null 时返回 0，非 null 时返回原值 */
    @Test
    public void testGetThinking() {
        TokenData data = new TokenData();
        Assert.assertEquals(Integer.valueOf(0), data.getThinking());
        data = new TokenData(100, null, null, null);
        Assert.assertEquals(Integer.valueOf(100), data.getThinking());
    }

    /** getInput：null 时返回 0，非 null 时返回原值 */
    @Test
    public void testGetInput() {
        TokenData data = new TokenData(null, 200, null, null);
        Assert.assertEquals(Integer.valueOf(200), data.getInput());
        data = new TokenData();
        Assert.assertEquals(Integer.valueOf(0), data.getInput());
    }

    /** getToken：null 时返回 0，非 null 时返回原值 */
    @Test
    public void testGetToken() {
        TokenData data = new TokenData(null, null, 300, null);
        Assert.assertEquals(Integer.valueOf(300), data.getTotal());
        data = new TokenData(null, null, null, null);
        Assert.assertEquals(Integer.valueOf(0), data.getTotal());
    }

    /** getCache：null 时返回 0，非 null 时返回原值 */
    @Test
    public void testGetCache() {
        TokenData data = new TokenData(null, null, null, 400);
        Assert.assertEquals(Integer.valueOf(400), data.getCache());
        data = new TokenData();
        Assert.assertEquals(Integer.valueOf(0), data.getCache());
    }

    /** hasData：token 为 null 或 0 时为 false，token > 0 时为 true */
    @Test
    public void testHasData() {
        TokenData data = new TokenData(null, null, null, null);
        Assert.assertFalse(data.hasData());
        data = new TokenData(null, null, 0, null);
        Assert.assertFalse(data.hasData());
        data = new TokenData(null, null, 1, null);
        Assert.assertTrue(data.hasData());
    }

    /** Builder：覆盖 thinking, input, token, cache 四个属性 */
    @Test
    public void testBuilder() {
        TokenData data = TokenData.builder()
                .thinking(11)
                .input(22)
                .total(33)
                .cache(44)
                .build();
        Assert.assertEquals(Integer.valueOf(11), data.getThinking());
        Assert.assertEquals(Integer.valueOf(22), data.getInput());
        Assert.assertEquals(Integer.valueOf(33), data.getTotal());
        Assert.assertEquals(Integer.valueOf(44), data.getCache());
        Assert.assertTrue(data.hasData());
    }

    /** Builder 部分设置：未设置的属性为 null，getter 返回 0 */
    @Test
    public void testBuilder_partial() {
        TokenData data = TokenData.builder()
                .thinking(1)
                .total(2)
                .build();
        Assert.assertEquals(Integer.valueOf(1), data.getThinking());
        Assert.assertEquals(Integer.valueOf(0), data.getInput());
        Assert.assertEquals(Integer.valueOf(2), data.getTotal());
        Assert.assertEquals(Integer.valueOf(0), data.getCache());
        Assert.assertTrue(data.hasData());
    }
}
