package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.rag.meta.RagMeta;
import ai.open.right.workflow.flow.llm.rag.meta.RagMetaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

/**
 * RagMeta 单测：覆盖 rag、metadata、buildMetadata、InitConfig 及内部类。
 */
public class RagMetaTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagMeta.class.getConstructor().newInstance();
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagMeta.InitConfig.class.getConstructor().newInstance();
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testRag() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        RagMeta ragMeta = new RagMeta();
        ragMeta.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
    }

    @Test
    public void testRagWithEmptyMetadata() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        RagMeta ragMeta = new RagMeta();
        ragMeta.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
    }

    @Test
    public void testRagWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .build();
        RagMeta meta = new RagMeta();
        meta.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, meta.rag(ragConfig, ragData));
    }

    @Test
    public void testRagWithMapXmlMode() throws Exception {
        Map<String, Object> meta = new HashMap<>();
        meta.put("Profile1", "DiscordConfigTest");
        meta.put("Profile2", Collections.singletonMap("Profile3", "C"));
        LLMQuery query = ObjectBuilder.buildLLMQuery(meta);
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setOverride(true);
        ragConfig.setMode(RagConfig.MODE_XML);
        RagMeta ragMeta = new RagMeta();
        ragMeta.rag(ragConfig, ragData);
        Assert.assertEquals("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Metadata xmlns=\"\"><Item><content>C</content><mcode>Profile2</mcode></Item><Item>DiscordConfigTest<mcode>Profile1</mcode></Item></Metadata>", query.getQuery());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagMeta.InitConfig service = new RagMeta.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagMeta empty = service.ragMeta();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }

    // ---------- metadata() 覆盖 ----------

    @Test
    public void testMetadataWhenEmpty() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagMetaConfig(new RagMetaConfig());
        RagData ragData = RagData.builder().config(new LLMConfig()).query(ObjectBuilder.buildLLMQuery()).build();
        Method m = RagMeta.class.getDeclaredMethod("metadata", RagConfig.class, RagData.class, Map.class);
        m.setAccessible(true);
        Map<String, Object> empty = Collections.emptyMap();
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(ragMeta, ragConfig, ragData, empty);
        Assert.assertSame(empty, result);
    }

    @Test
    public void testMetadataWhenNull() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagMetaConfig(new RagMetaConfig());
        RagData ragData = RagData.builder().config(new LLMConfig()).query(ObjectBuilder.buildLLMQuery()).build();
        Method m = RagMeta.class.getDeclaredMethod("metadata", RagConfig.class, RagData.class, Map.class);
        m.setAccessible(true);
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(ragMeta, ragConfig, ragData, (Map<String, Object>) null);
        Assert.assertNull(result);
    }

    @Test
    public void testMetadataWhenNoRagMeta() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setRagMetaConfig(null);
        RagData ragData = RagData.builder().config(new LLMConfig()).query(ObjectBuilder.buildLLMQuery()).build();
        Method m = RagMeta.class.getDeclaredMethod("metadata", RagConfig.class, RagData.class, Map.class);
        m.setAccessible(true);
        Map<String, Object> input = new HashMap<>();
        input.put("a", "1");
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(ragMeta, ragConfig, ragData, input);
        Assert.assertSame(input, result);
    }

    @Test
    public void testMetadataFiltersByWhiteList() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        RagMetaConfig metaConfig = new RagMetaConfig();
        metaConfig.setWhiteList(Arrays.asList("allow.*", "exact"));
        ragConfig.setRagMetaConfig(metaConfig);
        RagData ragData = RagData.builder().config(new LLMConfig()).query(ObjectBuilder.buildLLMQuery()).build();
        Method m = RagMeta.class.getDeclaredMethod("metadata", RagConfig.class, RagData.class, Map.class);
        m.setAccessible(true);
        Map<String, Object> input = new HashMap<>();
        input.put("allow.1", "v1");
        input.put("exact", "v2");
        input.put("forbidden", "v3");
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(ragMeta, ragConfig, ragData, input);
        Assert.assertEquals(2, result.size());
        Assert.assertEquals("v1", result.get("allow.1"));
        Assert.assertEquals("v2", result.get("exact"));
        Assert.assertFalse(result.containsKey("forbidden"));
    }

    @Test
    public void testMetadataFiltersByBlackList() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        RagMetaConfig metaConfig = new RagMetaConfig();
        metaConfig.setBlackList(Arrays.asList("no.*", "block"));
        ragConfig.setRagMetaConfig(metaConfig);
        RagData ragData = RagData.builder().config(new LLMConfig()).query(ObjectBuilder.buildLLMQuery()).build();
        Method m = RagMeta.class.getDeclaredMethod("metadata", RagConfig.class, RagData.class, Map.class);
        m.setAccessible(true);
        Map<String, Object> input = new HashMap<>();
        input.put("no.1", "v1");
        input.put("block", "v2");
        input.put("ok", "v3");
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) m.invoke(ragMeta, ragConfig, ragData, input);
        Assert.assertEquals(1, result.size());
        Assert.assertEquals("v3", result.get("ok"));
    }

    // ---------- buildMetadata() 覆盖 ----------

    @Test
    public void testBuildMetadataJsonMode() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_JSON);
        Map<String, Object> meta = new HashMap<>();
        meta.put("k", "v");
        LLMQuery query = ObjectBuilder.buildLLMQuery(meta);
        RagData ragData = RagData.builder().config(new LLMConfig()).query(query).build();
        Method m = RagMeta.class.getDeclaredMethod("buildMetadata", RagConfig.class, RagData.class);
        m.setAccessible(true);
        Object result = m.invoke(ragMeta, ragConfig, ragData);
        Assert.assertTrue(result instanceof Map);
        @SuppressWarnings("unchecked")
        Map<String, Object> map = (Map<String, Object>) result;
        Assert.assertEquals("v", map.get("k"));
    }

    @Test
    public void testBuildMetadataEmptyMetadata() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_JSON);
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder().config(new LLMConfig()).query(query).build();
        Method m = RagMeta.class.getDeclaredMethod("buildMetadata", RagConfig.class, RagData.class);
        m.setAccessible(true);
        Object result = m.invoke(ragMeta, ragConfig, ragData);
        Assert.assertTrue(result instanceof Map);
        Assert.assertTrue(((Map<?, ?>) result).isEmpty());
    }

    @Test
    public void testBuildMetadataXmlMode() throws Exception {
        RagMeta ragMeta = new RagMeta();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_XML);
        Map<String, Object> meta = new HashMap<>();
        meta.put("m1", "c1");
        LLMQuery query = ObjectBuilder.buildLLMQuery(meta);
        RagData ragData = RagData.builder().config(new LLMConfig()).query(query).build();
        Method m = RagMeta.class.getDeclaredMethod("buildMetadata", RagConfig.class, RagData.class);
        m.setAccessible(true);
        Object result = m.invoke(ragMeta, ragConfig, ragData);
        Assert.assertTrue(result instanceof RagMeta.LLMMetadataPrompts);
        RagMeta.LLMMetadataPrompts prompts = (RagMeta.LLMMetadataPrompts) result;
        Assert.assertNotNull(prompts.getItem());
        Assert.assertEquals(1, prompts.getItem().size());
        Assert.assertEquals("m1", prompts.getItem().get(0).getMcode());
        Assert.assertEquals("c1", prompts.getItem().get(0).getContent());
    }

    // ---------- 内部类覆盖 ----------

    @Test
    public void testLLMMetadataPromptsAdd() {
        RagMeta.LLMMetadataPrompts prompts = new RagMeta.LLMMetadataPrompts();
        RagMeta.LLMItemPrompts input = RagMeta.LLMItemPrompts.builder().mcode("x").content("y").build();
        RagMeta.LLMMetadataPrompts same = prompts.add(input);
        Assert.assertSame(prompts, same);
        Assert.assertEquals(1, prompts.getItem().size());
        Assert.assertEquals("x", prompts.getItem().get(0).getMcode());
        Assert.assertEquals("y", prompts.getItem().get(0).getContent());
    }

    @Test
    public void testLLMInputPromptsBuilder() {
        RagMeta.LLMItemPrompts p = RagMeta.LLMItemPrompts.builder()
                .mcode("code")
                .content("content")
                .build();
        Assert.assertEquals("code", p.getMcode());
        Assert.assertEquals("content", p.getContent());
    }
}
