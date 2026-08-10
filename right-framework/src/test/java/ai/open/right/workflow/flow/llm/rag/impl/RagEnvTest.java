package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;

public class RagEnvTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagEnv.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagEnv.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagEnv ragEnv = new RagEnv();
        ragEnv.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertTrue(ragData.getPrompt().endsWith("</Env>"));
    }

    @Test
    public void testWithNotMathPrompt() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        RagEnv ragEnv = new RagEnv();
        ragEnv.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
    }

    @Test
    public void testWithJsonMode() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQueryWithEmptyMetadata();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_JSON);
        RagEnv ragEnv = new RagEnv();
        ragEnv.rag(ragConfig, ragData);
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertTrue(ragData.getPrompt().endsWith("}"));
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .build();
        RagEnv ragEnv = new RagEnv();
        ragEnv.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, ragEnv.rag(ragConfig, ragData));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagEnv.InitConfig service = new RagEnv.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagEnv empty = service.ragEnv();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }

    @Test
    public void testWithConfig() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        List<String> keys = new ArrayList<>();
        keys.add(System.getenv().entrySet().iterator().next().getKey());
        ragConfig.setEnvironment(keys);
        Assert.assertEquals(keys, ragConfig.getEnvironment());
        ragConfig.setReplace("#key");
        ragConfig.setMode(RagConfig.MODE_XML);
        RagEnv ragEnv = new RagEnv();
        ragEnv.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", query.getQuery());
        Assert.assertTrue(ragData.getPrompt().startsWith("UNKNOWN"));
        Assert.assertTrue(ragData.getPrompt().endsWith("</Env>"));
    }
}
