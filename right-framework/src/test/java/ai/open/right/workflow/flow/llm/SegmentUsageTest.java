package ai.open.right.workflow.flow.llm;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.token.TokenData;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.Map;

/**
 * Test for {@link SegmentUsage}
 * 注释约定 JSON 格式：{"prompt_tokens":5000,"completion_tokens":100,"total_tokens":5100,"prompt_tokens_details":{"cached_tokens":3840},"completion_tokens_details":{"reasoning_tokens":0,...}}
 * 不存在的字段（如 completion_tokens）对比可忽略
 */
public class SegmentUsageTest {

    @Test
    public void test1() {
        // 测试默认值及 Builder 逻辑
        SegmentUsage segmentUsage = new SegmentUsage();
        Assertions.assertEquals(0, segmentUsage.getCache());
        Assertions.assertEquals(0, segmentUsage.getTotal());
    }

    @Test
    public void test2() {
        // 测试 Builder 赋值逻辑
        SegmentUsage segmentUsage = new SegmentUsage(TokenData.builder()
                .total(10).cache(20).build());
        Assertions.assertEquals(20, segmentUsage.getCache());
        Assertions.assertEquals(10, segmentUsage.getTotal());
    }

    @Test
    public void testSetters() {
        // 测试 setter，确保 getCache/getTotal 与注释约定一致
        SegmentUsage segmentUsage = new SegmentUsage();
        SegmentUsage segmentUsage2 = new SegmentUsage(TokenData.builder().cache(30)
                .total(40).build());
        segmentUsage.addUsage(segmentUsage2);
        Assertions.assertEquals(30, segmentUsage.getCache());
        segmentUsage.addUsage(segmentUsage2);
        Assertions.assertEquals(80, segmentUsage.getTotal());
    }

    @Test
    public void testConstructorFromTokenData() {
        // 测试 TokenData 构造：reasoning_tokens -> getThinking，cached_tokens -> getCache，token -> total，input -> getInput
        TokenData tokenData = TokenData.builder()
                .thinking(5)
                .cache(3840)
                .total(5100)
                .input(5000)
                .build();
        SegmentUsage segmentUsage = new SegmentUsage(tokenData);
        Assertions.assertEquals(5, segmentUsage.getThinking());
        Assertions.assertEquals(3840, segmentUsage.getCache());
        Assertions.assertEquals(5100, segmentUsage.getTotal());
        Assertions.assertEquals(5000, segmentUsage.getInput());
    }

    @Test
    public void testJsonFormatMatchesComment() throws Exception {
        // 注释格式：prompt_tokens, total_tokens, prompt_tokens_details.cached_tokens, completion_tokens_details.reasoning_tokens
        SegmentUsage segmentUsage = new SegmentUsage(TokenData.builder()
                .cache(3840).total(5100).input(5000).thinking(0).build());
        String json = JsonUtils.write(segmentUsage);
        Map<String, Object> parsed = JsonUtils.read(json, Map.class);
        Assertions.assertTrue(parsed.containsKey("prompt_tokens"), "JSON 应包含 prompt_tokens");
        Assertions.assertTrue(parsed.containsKey("total_tokens"), "JSON 应包含 total_tokens");
        Assertions.assertTrue(parsed.containsKey("prompt_tokens_details"), "JSON 应包含 prompt_tokens_details");
        Assertions.assertTrue(parsed.containsKey("completion_tokens_details"), "JSON 应包含 completion_tokens_details");
        Assertions.assertEquals(5000, ((Number) parsed.get("prompt_tokens")).intValue());
        Assertions.assertEquals(5100, ((Number) parsed.get("total_tokens")).intValue());
        @SuppressWarnings("unchecked")
        Map<String, Object> promptDetails = (Map<String, Object>) parsed.get("prompt_tokens_details");
        Assertions.assertNotNull(promptDetails);
        Assertions.assertTrue(promptDetails.containsKey("cached_tokens"), "prompt_tokens_details 应包含 cached_tokens");
        Assertions.assertEquals(3840, ((Number) promptDetails.get("cached_tokens")).intValue());
        @SuppressWarnings("unchecked")
        Map<String, Object> completionDetails = (Map<String, Object>) parsed.get("completion_tokens_details");
        Assertions.assertNotNull(completionDetails);
        Assertions.assertTrue(completionDetails.containsKey("reasoning_tokens"), "completion_tokens_details 应包含 reasoning_tokens");
    }

    /**
     * 覆盖 SegmentUsage#addUsage(LLMUsage)：累加 thinking、cache、total、input
     */
    @Test
    public void testAddUsage_accumulatesThinkingCacheTotalInput() {
        SegmentUsage base = new SegmentUsage(TokenData.builder()
                .thinking(1).cache(2).total(10).input(8).build());
        LLMUsage other = new SegmentUsage(TokenData.builder()
                .thinking(3).cache(5).total(20).input(12).build());
        base.addUsage(other);
        Assertions.assertEquals(4, base.getThinking());
        Assertions.assertEquals(7, base.getCache());
        Assertions.assertEquals(30, base.getTotal());
        Assertions.assertEquals(20, base.getInput());
    }

    /**
     * 覆盖 SegmentUsage#addUsage(LLMUsage)：入参 getThinking/getCache 为 null 时按 0 累加
     */
    @Test
    public void testAddUsage_nullThinkingAndCache_treatedAsZero() {
        SegmentUsage base = new SegmentUsage(TokenData.builder()
                .thinking(2).cache(3).total(5).input(4).build());
        LLMUsage other = new LLMUsage() {
            @Override
            public Integer getThinking() { return null; }
            @Override
            public Integer getCache() { return null; }
            @Override
            public Integer getTotal() { return 10; }
            @Override
            public Integer getInput() { return 6; }
            @Override
            public void addUsage(LLMUsage usage) {}
        };
        base.addUsage(other);
        Assertions.assertEquals(2, base.getThinking());
        Assertions.assertEquals(3, base.getCache());
        Assertions.assertEquals(15, base.getTotal());
        Assertions.assertEquals(10, base.getInput());
    }
}
