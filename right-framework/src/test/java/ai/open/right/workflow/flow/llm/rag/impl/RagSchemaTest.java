package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.collections.MapUtils;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class RagSchemaTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagSchema.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagSchema.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testQueryMetadataSchemaIgnoredWhenAdditionalEmpty() throws Exception {
        Map<String, Object> meta = new HashMap<>();
        meta.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("from", "metadata"));
        LLMQuery query = ObjectBuilder.buildLLMQuery(meta);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of());
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.init(llmConfig);
        ragConfig.setReplace("#key");
        RagSchema ragSchema = new RagSchema();
        ragSchema.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN ", ragData.getPrompt());
        Assert.assertFalse(ragData.getPrompt().contains("from"));
    }

    @Test
    public void testSchemaFromAdditionalWhenQueryMetadataAlsoPresent() throws Exception {
        Map<String, Object> meta = new HashMap<>();
        meta.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("winner", "metadata"));
        LLMQuery query = ObjectBuilder.buildLLMQuery(meta);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of(ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("winner", "additional")));
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.init(llmConfig);
        ragConfig.setReplace("#key");
        RagSchema ragSchema = new RagSchema();
        ragSchema.rag(ragConfig, ragData);
        Assert.assertTrue(ragData.getPrompt().contains("\"winner\":\"additional\""));
        Assert.assertFalse(ragData.getPrompt().contains("metadata"));
    }

    @Test
    public void test() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of(ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("A", "B")));
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.init(llmConfig);
        ragConfig.setReplace("#key");
        RagSchema ragSchema = new RagSchema();
        ragSchema.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertEquals("UNKNOWN \n" +
                "```JSON SCHEMA\n" +
                "{\"A\":\"B\"}\n" +
                "```\n", ragData.getPrompt());
    }

    @Test
    public void testNull() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        LLMConfig llmConfig = new LLMConfig();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.init(llmConfig);
        ragConfig.setReplace("#key");
        RagSchema ragSchema = new RagSchema();
        ragSchema.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertEquals("UNKNOWN ", ragData.getPrompt());
    }

    @Test
    public void testEmpty() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of());
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.init(llmConfig);
        ragConfig.setReplace("#key");
        RagSchema ragSchema = new RagSchema();
        ragSchema.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertEquals("UNKNOWN ", ragData.getPrompt());
    }

    /** schema 取自 ragData.getConfig().getAdditional()，与 ragConfig.getLlmConfig() 可不一致 */
    @Test
    public void testBuildSchemaReadsRagDataConfigAdditionalOnly() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig onRagConfig = new LLMConfig();
        onRagConfig.setAdditional(ImmutableMap.of(ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("src", "ragConfig")));
        config.setLlmConfig(onRagConfig);
        LLMConfig onRagData = new LLMConfig();
        onRagData.setAdditional(ImmutableMap.of());
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(onRagData).query(query).prompt("p").build();
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }

    @Test
    public void testBuildSchemaBranchSchemaPresentFromAdditional() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of(ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("k", "v")));
        config.setLlmConfig(llmConfig);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(llmConfig).query(query).prompt("p").build();
        Object out = service.buildSchema(config, ragData);
        Assert.assertEquals("\n" +
                "```JSON SCHEMA\n" +
                "{\"k\":\"v\"}\n" +
                "```\n", out);
    }

    @Test
    public void testBuildSchemaQueryMetadataIgnoredWithoutAdditionalSchema() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        Map<String, Object> meta = new HashMap<>();
        meta.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("m", "1"));
        LLMQuery query = ObjectBuilder.buildLLMQuery(meta);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of());
        config.setLlmConfig(llmConfig);
        RagData ragData = RagData.builder().config(llmConfig).query(query).prompt("p").build();
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }

    @Test
    public void testBuildSchemaBranchSchemaAbsent() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of());
        config.setLlmConfig(llmConfig);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(llmConfig).query(query).prompt("p").build();
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }

    @Test
    public void testWithNotMathPrompt() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of(ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("A", "B")));
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.init(llmConfig);
        ragConfig.setReplace("#key");
        RagEnv ragEnv = new RagEnv();
        ragEnv.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertEquals("UNKNOWN", ragData.getPrompt());
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .build();
        RagSchema ragSchema = new RagSchema();
        ragSchema.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, ragSchema.rag(ragConfig, ragData));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagSchema.InitConfig service = new RagSchema.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagSchema empty = service.ragSchema();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }

    @Test
    public void testBuildSchemaNullConfig() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig ragDataConfig = new LLMConfig();
        ragDataConfig.setAdditional(null);
        LLMConfig ragConfigLlm = new LLMConfig();
        ragConfigLlm.setAdditional(ImmutableMap.of(ProviderRequestService.KEY_RESPONSE_SCHEMA, ImmutableMap.of("only", "onRagConfig")));
        config.setLlmConfig(ragConfigLlm);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(ragDataConfig).query(query).prompt("p").build();
        Assert.assertNull(MapUtils.getObject(ragData.getConfig().getAdditional(), ProviderRequestService.KEY_RESPONSE_SCHEMA));
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }

    /** MapUtils.getObject(ragData.getConfig().getAdditional(), KEY_RESPONSE_SCHEMA) 为 null：additional 为 null */
    @Test
    public void testBuildSchemaSchemaNullWhenAdditionalMapIsNull() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(null);
        config.setLlmConfig(llmConfig);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(llmConfig).query(query).prompt("p").build();
        Assert.assertNull(MapUtils.getObject(ragData.getConfig().getAdditional(), ProviderRequestService.KEY_RESPONSE_SCHEMA));
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }

    /** MapUtils.getObject(ragData.getConfig().getAdditional(), KEY_RESPONSE_SCHEMA) 为 null：未配置 response_schema 键 */
    @Test
    public void testBuildSchemaSchemaNullWhenResponseSchemaKeyAbsent() throws Exception {
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(ImmutableMap.of("other_key", "x"));
        config.setLlmConfig(llmConfig);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(llmConfig).query(query).prompt("p").build();
        Assert.assertNull(MapUtils.getObject(ragData.getConfig().getAdditional(), ProviderRequestService.KEY_RESPONSE_SCHEMA));
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }

    /** MapUtils.getObject(ragData.getConfig().getAdditional(), KEY_RESPONSE_SCHEMA) 为 null：键存在且值为 null */
    @Test
    public void testBuildSchemaSchemaNullWhenResponseSchemaValueIsNull() throws Exception {
        Map<String, Object> additional = new HashMap<>();
        additional.put(ProviderRequestService.KEY_RESPONSE_SCHEMA, null);
        RagSchema service = new RagSchema();
        RagConfig config = new RagConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setAdditional(additional);
        config.setLlmConfig(llmConfig);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder().config(llmConfig).query(query).prompt("p").build();
        Assert.assertNull(MapUtils.getObject(ragData.getConfig().getAdditional(), ProviderRequestService.KEY_RESPONSE_SCHEMA));
        Assert.assertEquals("", service.buildSchema(config, ragData));
    }
}
