package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallResponse;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.lang.reflect.Method;
import java.util.Map;

public class GooglePartTest {

    /** GooglePart(LLMFunCallResponse)：response 为非 JSON 明文时，functionResponse.response 为 Map 含 content。 */
    @Test
    public void testConstructorWithLLMFunCallResponse_PlainText() throws Exception {
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder()
                .name("tool_name")
                .response("tool_result")
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallResponse);
        Assert.assertNotNull(part.getFunctionResponse());
        Assert.assertEquals("tool_name", part.getFunctionResponse().get("name"));
        Object responseVal = part.getFunctionResponse().get("response");
        Assert.assertNotNull(responseVal);
        Assert.assertTrue(responseVal instanceof java.util.Map);
        Assert.assertEquals("tool_result", ((java.util.Map<?, ?>) responseVal).get("content"));
    }

    /** GooglePart(LLMFunCallResponse)：response 为 JSON 字符串时，functionResponse.response 为解析后的 Map。 */
    @Test
    public void testConstructorWithLLMFunCallResponse_JsonLike() throws Exception {
        String jsonBody = "{\"key\":\"val\",\"num\":1}";
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder()
                .name("fn")
                .response(jsonBody)
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallResponse);
        Assert.assertEquals("fn", part.getFunctionResponse().get("name"));
        Object responseVal = part.getFunctionResponse().get("response");
        Assert.assertNotNull(responseVal);
        Assert.assertTrue(responseVal instanceof java.util.Map);
        Assert.assertEquals("val", ((java.util.Map<?, ?>) responseVal).get("key"));
        Assert.assertEquals(1, ((java.util.Map<?, ?>) responseVal).get("num"));
    }

    /** GooglePart(LLMFunCallResponse)：response 为 null 时，content 落为 ""，不抛 NPE。 */
    @Test
    public void testConstructorWithLLMFunCallResponse_NullResponse() throws Exception {
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder()
                .name("tool_name")
                .response(null)
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallResponse);
        Assert.assertEquals("tool_name", part.getFunctionResponse().get("name"));
        Object responseVal = part.getFunctionResponse().get("response");
        Assert.assertNotNull(responseVal);
        Assert.assertEquals("", ((java.util.Map<?, ?>) responseVal).get("content"));
    }

    /** GooglePart(LLMFunCallResponse)：name 为 null 时仍写入。 */
    @Test
    public void testConstructorWithLLMFunCallResponse_NullName() throws Exception {
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder()
                .name(null)
                .response("body")
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallResponse);
        Assert.assertNull(part.getFunctionResponse().get("name"));
        Assert.assertEquals("body", ((java.util.Map<?, ?>) part.getFunctionResponse().get("response")).get("content"));
    }

    /** GooglePart(LLMFunCallResponse)：JSON 形似但解析异常时，覆盖 catch (Exception e) { log.debug(e.getMessage(), e); } 分支，response 仍为 null，落为 content 分支。 */
    @Test
    public void testConstructorWithLLMFunCallResponse_JsonLikeButParseFails() throws Exception {
        String invalidJson = "{invalid";
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder()
                .name("n")
                .response(invalidJson)
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallResponse);
        Assert.assertEquals("n", part.getFunctionResponse().get("name"));
        Object responseVal = part.getFunctionResponse().get("response");
        Assert.assertNotNull(responseVal);
        Assert.assertEquals(invalidJson, ((java.util.Map<?, ?>) responseVal).get("content"));
    }

    /** 覆盖 catch (Exception e) { log.debug(e.getMessage(), e); }：body 为合法 JSON 但非 Map（如数组），JsonUtils.read(body, Map.class) 抛异常后走 catch，最终落 content 分支。 */
    @Test
    public void testConstructorWithLLMFunCallResponse_JsonLikeButNotMapCausesCatch() throws Exception {
        String jsonArray = "[]";
        LLMFunCallResponse llmFunCallResponse = ProviderFunCallResponse.builder()
                .name("tool")
                .response(jsonArray)
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallResponse);
        Assert.assertEquals("tool", part.getFunctionResponse().get("name"));
        Object responseVal = part.getFunctionResponse().get("response");
        Assert.assertNotNull(responseVal);
        Assert.assertEquals(jsonArray, ((java.util.Map<?, ?>) responseVal).get("content"));
    }

    @Test
    public void test1() throws Exception {
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart("HELLO WORLD");
        Assert.assertEquals("HELLO WORLD", part.getText());
    }

    @Test
    public void test2() throws Exception {
        LLMFunCallRequest llmFunCallRequest = ProviderFunCallRequest.builder()
                .refer(ImmutableMap.of("thoughtSignature", "value"))
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallRequest);
        Assert.assertEquals("value", part.getThoughtSignature());
    }

    @Test
    public void test3() throws Exception {
        LLMFunCallRequest llmFunCallRequest = ProviderFunCallRequest.builder()
                .build();
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart(llmFunCallRequest);
        Assert.assertEquals("c2lnbmF0dXJlLTQ3NzQ4MTdiLTFhNGItNDU5MC04MWRmLWY4ZjZkOWY0NzM3YQ==", part.getThoughtSignature());
    }

    /** buildJson：args 为 Map 时直接返回该 Map。 */
    @Test
    public void testBuildJsonWhenArgsIsMap() throws Exception {
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart("t");
        Method m = GoogleRouter.GoogleMessage.GooglePart.class.getDeclaredMethod("buildJson", Object.class);
        m.setAccessible(true);
        Map<String, Object> input = ImmutableMap.of("k", "v", "a", 1);
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(part, input);
        Assert.assertSame(input, result);
        Assert.assertEquals("v", result.get("k"));
        Assert.assertEquals(1, result.get("a"));
    }

    /** buildJson：args 非 Map 但 JsonUtils.transfer 可转为 Map 时返回转换结果。 */
    @Test
    public void testBuildJsonWhenArgsTransferToMap() throws Exception {
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart("t");
        Method m = GoogleRouter.GoogleMessage.GooglePart.class.getDeclaredMethod("buildJson", Object.class);
        m.setAccessible(true);
        String jsonLike = "{\"x\":\"y\",\"n\":2}";
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(part, jsonLike);
        Assert.assertNotNull(result);
        Assert.assertEquals("y", result.get("x"));
        Assert.assertEquals(2, result.get("n"));
    }

    /** buildJson：转换抛异常时兜底返回 ImmutableMap.of("args", JsonUtils.write(args))。 */
    @Test
    public void testBuildJsonWhenTransferFailsFallback() throws Exception {
        GoogleRouter.GoogleMessage.GooglePart part = new GoogleRouter.GoogleMessage.GooglePart("t");
        Method m = GoogleRouter.GoogleMessage.GooglePart.class.getDeclaredMethod("buildJson", Object.class);
        m.setAccessible(true);
        byte[] args = new byte[]{1, 2, 3};
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(part, args);
        Assert.assertNotNull(result);
        Assert.assertEquals(1, result.size());
        Assert.assertTrue(result.containsKey("args"));
        Assert.assertNotNull(result.get("args"));
    }
}
