package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class RagUserTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagUser.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagUser.InitConfig.class.getConstructor(null).newInstance(null);
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
        RagUser ragUser = new RagUser();
        ragUser.rag(ragConfig, ragData);
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagUser user = new RagUser();
        user.setNotifierService(notifierManager);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_XML);
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, user.rag(ragConfig, ragData));
    }

    @Test
    public void testWithJson() throws Exception {
        RagUser user = new RagUser();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_JSON);
        UserContext userContext = ObjectBuilder.buildEmpty();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setUserContext(userContext);
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Assert.assertEquals(userContext, user.buildUserContext(ragConfig, ragData));
    }

    @Test
    public void testWithXml() throws Exception {
        RagUser user = new RagUser();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setMode(RagConfig.MODE_XML);
        UserContext userContext = ObjectBuilder.buildEmpty();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.setUserContext(userContext);
        RagData ragData = RagData.builder()
                .query(llmQuery)
                .build();
        Assert.assertEquals("{\"language\":\"UNKNOWN\",\"system\":\"UNKNOWN\",\"device\":\"UNKNOWN\",\"region\":\"UNKNOWN\",\"brand\":\"UNKNOWN\",\"model\":\"UNKNOWN\"}", JsonUtils.write(user.buildUserContext(ragConfig, ragData)));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagUser.InitConfig service = new RagUser.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagUser empty = service.ragUser();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }
}
