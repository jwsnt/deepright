package ai.open.right.workflow.flow.llm.rag.query;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.digest.RagDigest;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class RagPreviousQueryTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagPreviousQuery.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagPreviousQuery.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testQuery() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("HI#key HELLO WORLD")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        RagPreviousQuery ragPreviousQuery = new RagPreviousQuery();
        ragPreviousQuery.rag(ragConfig, ragData);
        Assert.assertEquals("HIPREVIOUS HELLO WORLD", ragData.getPrompt());
    }

    @Test
    public void testPrompt() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #KEY")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#KEY");
        RagPreviousQuery ragPreviousQuery = new RagPreviousQuery();
        ragPreviousQuery.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN PREVIOUS", ragData.getPrompt());
    }

    @Test
    public void testNotAllowed() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #KEY")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#KEY");
        RagPreviousQuery ragPreviousQuery = new RagPreviousQuery() {
            protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
                return false;
            }
        };
        ragPreviousQuery.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN #KEY", ragData.getPrompt());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagPreviousQuery.InitConfig service = new RagPreviousQuery.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagPreviousQuery empty = service.ragPreviousQuery();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }
}
