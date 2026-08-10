package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class RagDimensionTest {


    @Test
    public void testHashCode1() throws Exception {
        Object object = RagDimension.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagDimension.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void test() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("HELLO")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        RagDimension ragDimension = new RagDimension();
        ragDimension.rag(ragConfig, ragData);
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagDimension ragDimension = new RagDimension();
        ragDimension.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_XML);
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, ragDimension.rag(ragConfig, ragData));
    }

    @Test
    public void testWithJson() throws Exception {
        RagDimension ragDimension = new RagDimension();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_JSON);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Assert.assertEquals("{\"workflow\":\"UNKNOWN\",\"device\":\"UNKNOWN\",\"chat\":\"UNKNOWN\",\"biz\":\"UNKNOWN\"}", JsonUtils.write(ragDimension.buildDimension(ragConfig, ragData)));
    }

    @Test
    public void testWithXml() throws Exception {
        RagDimension ragDimension = new RagDimension();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_XML);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Assert.assertEquals("{\"workflow\":\"UNKNOWN\",\"device\":\"UNKNOWN\",\"chat\":\"UNKNOWN\",\"biz\":\"UNKNOWN\"}", JsonUtils.write(ragDimension.buildDimension(ragConfig, ragData)));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagDimension.InitConfig service = new RagDimension.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagDimension empty = service.ragDimension();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }
}
