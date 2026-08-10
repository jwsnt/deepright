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

public class RagOriginalQueryTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagOriginalQuery.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagOriginalQuery.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testQuery() throws Exception {
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        RagOriginalQuery ragOriginalQuery = new RagOriginalQuery();
        ragOriginalQuery.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWNORIGINAL", query.getQuery());
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
        RagOriginalQuery ragOriginalQuery = new RagOriginalQuery();
        ragOriginalQuery.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN ORIGINAL", ragData.getPrompt());
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
        RagOriginalQuery ragOriginalQuery = new RagOriginalQuery() {
            protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
                return false;
            }
        };
        ragOriginalQuery.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN #KEY", ragData.getPrompt());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagOriginalQuery.InitConfig service = new RagOriginalQuery.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Condition(10086);
        RagOriginalQuery empty = service.ragOriginalQuery();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertNotNull(empty);
    }
}
