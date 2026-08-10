package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.utils.JsonUtils;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class ProviderFunCallResponseTest {

    /** 覆盖 @NoArgsConstructor：直接调用无参构造，验证所有字段为 null，isValid() 为 false。 */
    @Test
    public void testNoArgsConstructor() {
        ProviderFunCallResponse response = new ProviderFunCallResponse();
        Assert.assertNull(response.getMetadata());
        Assert.assertNull(response.getResponse());
        Assert.assertNull(response.getModel());
        Assert.assertNull(response.getName());
        Assert.assertNull(response.getId());
        Assert.assertFalse(response.isValid());
    }

    // 测试无参构造及Getter/Setter
    @Test
    public void testNoArgsConstructorAndGetSet() {
        ProviderFunCallResponse response = ProviderFunCallResponse.builder().build();
        // 初始值验证
        Assert.assertNull(response.getMetadata());
        Assert.assertNull(response.getResponse());
        Assert.assertNull(response.getModel());
        Assert.assertNull(response.getName());
        Assert.assertNull(response.getId());
        // Setter验证
        Map<String, Object> meta = new HashMap<>();
        meta.put("key1", "value1");
        response.setMetadata(meta);
        response.setResponse("testResponse");
        response.setModel("gemini-pro");
        response.setName("testName");
        response.setId("testId");
        Assert.assertEquals(meta, response.getMetadata());
        Assert.assertEquals("value1", response.getMetadata().get("key1"));
        Assert.assertEquals("testResponse", response.getResponse());
        Assert.assertEquals("gemini-pro", response.getModel());
        Assert.assertEquals("testName", response.getName());
        Assert.assertEquals("testId", response.getId());
    }

    // 测试Builder模式
    @Test
    public void testBuilder() {
        Map<String, Object> meta = new HashMap<>();
        meta.put("k", "v");
        ProviderFunCallResponse response = ProviderFunCallResponse.builder()
                .metadata(meta)
                .response("builderResponse")
                .model("m1")
                .name("builderName")
                .id("builderId")
                .build();
        Assert.assertEquals(meta, response.getMetadata());
        Assert.assertEquals("v", response.getMetadata().get("k"));
        Assert.assertEquals("builderResponse", response.getResponse());
        Assert.assertEquals("m1", response.getModel());
        Assert.assertEquals("builderName", response.getName());
        Assert.assertEquals("builderId", response.getId());
    }

    // 测试toString方法
    @Test
    public void testToString() {
        Map<String, Object> meta = Collections.singletonMap("m", "metaVal");
        ProviderFunCallResponse response = ProviderFunCallResponse.builder()
                .metadata(meta)
                .response("toStringResp")
                .name("toStringName")
                .id("toStringId")
                .build();
        String str = response.toString();
        Assert.assertTrue(str.contains("metadata=" + meta.toString()));
        Assert.assertTrue(str.contains("response=toStringResp"));
        ProviderFunCallResponse withModel = ProviderFunCallResponse.builder()
                .name("n")
                .model("mdl")
                .response("r")
                .build();
        Assert.assertTrue(withModel.toString().contains("model=mdl"));
        Assert.assertTrue(str.contains("name=toStringName"));
        Assert.assertTrue(str.contains("id=toStringId"));
        Assert.assertTrue(str.startsWith("ProviderFunCallResponse("));
        Assert.assertTrue(str.endsWith(")"));
    }

    // 测试isValid()：name有效场景
    @Test
    public void testIsValid_ValidName() {
        // 非空字符串
        ProviderFunCallResponse response1 = ProviderFunCallResponse.builder()
                .name("valid")
                .build();
        Assert.assertTrue(response1.isValid());
        // 含前后空格的非空白字符串
        ProviderFunCallResponse response2 = ProviderFunCallResponse.builder()
                .name("  validName  ")
                .build();
        Assert.assertTrue(response2.isValid());
    }

    // 测试isValid()：name无效场景
    @Test
    public void testIsValid_InvalidName() {
        // name为null
        ProviderFunCallResponse response1 = ProviderFunCallResponse.builder()
                .name(null)
                .build();
        Assert.assertFalse(response1.isValid());

        // name为空字符串
        ProviderFunCallResponse response2 = ProviderFunCallResponse.builder()
                .name("")
                .build();
        Assert.assertFalse(response2.isValid());

        // name为全空白（空格+制表符）
        ProviderFunCallResponse response3 = ProviderFunCallResponse.builder()
                .name("   \t  ")
                .build();
        Assert.assertFalse(response3.isValid());
    }

    // 验证isValid()与response字段无关
    @Test
    public void testIsValid_IndependentOfResponse() {
        // response为null但name有效
        ProviderFunCallResponse response1 = ProviderFunCallResponse.builder()
                .response(null)
                .name("valid")
                .build();
        Assert.assertTrue(response1.isValid());

        // response有值但name无效
        ProviderFunCallResponse response2 = ProviderFunCallResponse.builder()
                .response("nonNullResponse")
                .name("")
                .build();
        Assert.assertFalse(response2.isValid());
    }

    // metadata 为 null 时 getter 返回 null
    @Test
    public void testMetadata_Null() {
        ProviderFunCallResponse response = ProviderFunCallResponse.builder()
                .name("n")
                .build();
        Assert.assertNull(response.getMetadata());
    }

    // metadata 为空 Map 时 getter 返回空 Map，且可读
    @Test
    public void testMetadata_EmptyMap() {
        Map<String, Object> empty = Collections.emptyMap();
        ProviderFunCallResponse response = ProviderFunCallResponse.builder()
                .metadata(empty)
                .name("n")
                .build();
        Assert.assertNotNull(response.getMetadata());
        Assert.assertTrue(response.getMetadata().isEmpty());
    }

    // metadata 多键值对及 set 覆盖
    @Test
    public void testMetadata_MultipleEntriesAndSetOverwrite() {
        Map<String, Object> meta1 = new HashMap<>();
        meta1.put("a", 1);
        meta1.put("b", "two");
        ProviderFunCallResponse response = ProviderFunCallResponse.builder()
                .metadata(meta1)
                .name("n")
                .build();
        Assert.assertEquals(1, response.getMetadata().get("a"));
        Assert.assertEquals("two", response.getMetadata().get("b"));

        Map<String, Object> meta2 = new HashMap<>();
        meta2.put("x", "overwrite");
        response.setMetadata(meta2);
        Assert.assertEquals(meta2, response.getMetadata());
        Assert.assertEquals("overwrite", response.getMetadata().get("x"));
        Assert.assertNull(response.getMetadata().get("a"));
    }

    // isValid 与 metadata 无关
    @Test
    public void testIsValid_IndependentOfMetadata() {
        Map<String, Object> meta = Collections.singletonMap("k", "v");
        ProviderFunCallResponse withMetaValid = ProviderFunCallResponse.builder()
                .metadata(meta)
                .name("valid")
                .build();
        Assert.assertTrue(withMetaValid.isValid());

        ProviderFunCallResponse withMetaInvalid = ProviderFunCallResponse.builder()
                .metadata(meta)
                .name("")
                .build();
        Assert.assertFalse(withMetaInvalid.isValid());
    }

    /** putMetadata：metadata 为 null 时自动创建 Map 并写入键值。 */
    @Test
    public void testPutMetadata_WhenMetadataNull() {
        ProviderFunCallResponse response = new ProviderFunCallResponse();
        Assert.assertNull(response.getMetadata());
        response.putMetadata("A", "B");
        Assert.assertNotNull(response.getMetadata());
        Assert.assertEquals("B", response.getMetadata().get("A"));
    }

    /** putMetadata：多次调用，所有键值均生效。 */
    @Test
    public void testPutMetadata_MultiplePuts() {
        ProviderFunCallResponse response = new ProviderFunCallResponse();
        response.putMetadata("a", 1);
        response.putMetadata("b", "two");
        response.putMetadata("c", null);
        Assert.assertEquals(1, response.getMetadata().get("a"));
        Assert.assertEquals("two", response.getMetadata().get("b"));
        Assert.assertNull(response.getMetadata().get("c"));
        Assert.assertEquals(3, response.getMetadata().size());
    }

    /** putMetadata：已有 metadata 时在原有 Map 上追加，不覆盖。 */
    @Test
    public void testPutMetadata_WhenMetadataExists() {
        Map<String, Object> existing = new HashMap<>();
        existing.put("existing", "value");
        ProviderFunCallResponse response = ProviderFunCallResponse.builder()
                .metadata(existing)
                .name("n")
                .build();
        response.putMetadata("newKey", "newVal");
        Assert.assertEquals("value", response.getMetadata().get("existing"));
        Assert.assertEquals("newVal", response.getMetadata().get("newKey"));
        Assert.assertEquals(2, response.getMetadata().size());
    }

    /** model：JSON 往返与 OpenAiRouter 从 History 反序列化路径一致。 */
    @Test
    public void testModel_jsonRoundTrip() throws Exception {
        ProviderFunCallResponse built = ProviderFunCallResponse.builder()
                .name("tool_a")
                .id("id1")
                .response("ok")
                .model("claude-3")
                .build();
        String written = JsonUtils.write(built);
        Assert.assertTrue(written.contains("claude-3"));
        ProviderFunCallResponse back = JsonUtils.read(written, ProviderFunCallResponse.class);
        Assert.assertEquals("claude-3", back.getModel());

        String fromHistory = "{\"name\":\"x\",\"id\":\"i\",\"response\":\"r\",\"model\":\"vertex-gemini\"}";
        ProviderFunCallResponse parsed = JsonUtils.read(fromHistory, ProviderFunCallResponse.class);
        Assert.assertEquals("vertex-gemini", parsed.getModel());
    }
}