package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.utils.JsonUtils;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

/**
 * ProviderFunCallRequest 单测，覆盖 @NoArgsConstructor 及常用方法。
 */
public class ProviderFunCallRequestTest {

    /** 覆盖 @NoArgsConstructor：无参构造后各字段为默认值，timestamp 非 null。 */
    @Test
    public void testNoArgsConstructor() {
        ProviderFunCallRequest request = new ProviderFunCallRequest();
        Assert.assertNotNull(request.getCreated());
        Assert.assertNull(request.getMetadata());
        Assert.assertNull(request.getRefer());
        Assert.assertNull(request.getModel());
        Assert.assertNull(request.getArgs());
        Assert.assertNull(request.getName());
        Assert.assertNull(request.getId());
        Assert.assertFalse(request.isValid());
    }

    /** putMetadata：metadata 为 null 时自动创建 Map 并写入键值。 */
    @Test
    public void testPutMetadata_WhenMetadataNull() {
        ProviderFunCallRequest request = new ProviderFunCallRequest();
        Assert.assertNull(request.getMetadata());
        request.putMetadata("A", "B");
        Assert.assertNotNull(request.getMetadata());
        Assert.assertEquals("B", request.getMetadata().get("A"));
    }

    /** putMetadata：多次调用，所有键值均生效。 */
    @Test
    public void testPutMetadata_MultiplePuts() {
        ProviderFunCallRequest request = new ProviderFunCallRequest();
        request.putMetadata("a", 1);
        request.putMetadata("b", "two");
        request.putMetadata("c", null);
        Assert.assertEquals(1, request.getMetadata().get("a"));
        Assert.assertEquals("two", request.getMetadata().get("b"));
        Assert.assertNull(request.getMetadata().get("c"));
        Assert.assertEquals(3, request.getMetadata().size());
    }

    /** putMetadata：已有 metadata 时在原有 Map 上追加，不覆盖。 */
    @Test
    public void testPutMetadata_WhenMetadataExists() {
        Map<String, Object> existing = new HashMap<>();
        existing.put("existing", "value");
        ProviderFunCallRequest request = ProviderFunCallRequest.builder()
                .metadata(existing)
                .build();
        request.putMetadata("newKey", "newVal");
        Assert.assertEquals("value", request.getMetadata().get("existing"));
        Assert.assertEquals("newVal", request.getMetadata().get("newKey"));
        Assert.assertEquals(2, request.getMetadata().size());
    }

    /** Builder 及 isValid：name 有值则为 true。 */
    @Test
    public void testBuilderAndIsValid() {
        ProviderFunCallRequest request = ProviderFunCallRequest.builder()
                .name("n")
                .id("id1")
                .args("args")
                .build();
        Assert.assertTrue(request.isValid());
        Assert.assertEquals("n", request.getName());
        Assert.assertEquals("id1", request.getId());
        Assert.assertEquals("args", request.getArgs());
    }

    /** isValid：name 为空或 null 为 false。 */
    @Test
    public void testIsValid_EmptyName() {
        Assert.assertFalse(ProviderFunCallRequest.builder().name("").build().isValid());
        Assert.assertFalse(ProviderFunCallRequest.builder().name(null).build().isValid());
    }

    /** model：Builder、Setter 与历史 JSON 反序列化（OpenAiRouter 等从 History 读回）。 */
    @Test
    public void testModel_builderSetterAndJsonRoundTrip() throws Exception {
        ProviderFunCallRequest built = ProviderFunCallRequest.builder()
                .name("tool_x")
                .model("kimi-k2.5")
                .id("call-1")
                .build();
        Assert.assertEquals("kimi-k2.5", built.getModel());

        ProviderFunCallRequest manual = new ProviderFunCallRequest();
        manual.setName("n");
        manual.setModel("gpt-4o");
        Assert.assertEquals("gpt-4o", manual.getModel());

        String json = "{\"name\":\"t\",\"model\":\"deepseek-chat\",\"id\":\"i1\",\"args\":{}}";
        ProviderFunCallRequest fromHistory = JsonUtils.read(json, ProviderFunCallRequest.class);
        Assert.assertEquals("deepseek-chat", fromHistory.getModel());
        Assert.assertEquals("t", fromHistory.getName());
        ProviderFunCallRequest roundTrip = JsonUtils.read(JsonUtils.write(built), ProviderFunCallRequest.class);
        Assert.assertEquals(built.getModel(), roundTrip.getModel());
        Assert.assertEquals(built.getName(), roundTrip.getName());
    }
}
