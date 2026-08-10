package ai.open.right.workflow.flow.llm.rag.query;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.Prompt;
import ai.open.right.workflow.config.impl.PromptServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class RagStaticQueryTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagStaticQuery.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagStaticQuery.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testWithNoting() throws Exception {
        PromptServiceImpl promptManager = EasyMock.createMock(PromptServiceImpl.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        EasyMock.expect(prompt.getContent()).andReturn("").anyTimes();
        EasyMock.expect(promptManager.get(EasyMock.anyObject())).andReturn(prompt).anyTimes();
        EasyMock.replay(promptManager, prompt);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        RagStaticQuery ragQuery = new RagStaticQuery();
        ragQuery.setPromptService(promptManager);
        ragQuery.rag(ragConfig, ragData);
        Assert.assertEquals("UNKNOWN", ragData.getQuery().getQuery());
        EasyMock.verify(promptManager, prompt);
    }

    @Test
    public void testWithAppend() throws Exception {
        PromptServiceImpl promptManager = EasyMock.createMock(PromptServiceImpl.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        EasyMock.expect(prompt.getContent()).andReturn("#key HELLO WORLD").anyTimes();
        EasyMock.expect(promptManager.get(EasyMock.anyObject())).andReturn(prompt).anyTimes();
        EasyMock.replay(promptManager, prompt);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.setQuery("HI");
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setTemplate("ANYTHING");
        ragConfig.setReplace("#key");
        RagStaticQuery ragQuery = new RagStaticQuery();
        ragQuery.setPromptService(promptManager);
        ragQuery.rag(ragConfig, ragData);
        Assert.assertEquals("HI#key HELLO WORLD", ragData.getQuery().getQuery());
        EasyMock.verify(promptManager, prompt);
    }

    @Test
    public void testWithOverride() throws Exception {
        PromptServiceImpl promptManager = EasyMock.createMock(PromptServiceImpl.class);
        Prompt prompt = EasyMock.createMock(Prompt.class);
        EasyMock.expect(prompt.getContent()).andReturn("#key HELLO WORLD").anyTimes();
        EasyMock.expect(promptManager.get(EasyMock.anyObject())).andReturn(prompt).anyTimes();
        EasyMock.replay(promptManager, prompt);
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        query.setQuery("HI");
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setOverride(true);
        ragConfig.setTemplate("ANYTHING");
        ragConfig.setReplace("#key");
        RagStaticQuery ragQuery = new RagStaticQuery();
        ragQuery.setPromptService(promptManager);
        ragQuery.rag(ragConfig, ragData);
        Assert.assertEquals("HI HELLO WORLD", ragData.getQuery().getQuery());
        EasyMock.verify(promptManager, prompt);
    }

    @Test
    public void testInit() throws Exception {
        PromptServiceImpl promptManager = EasyMock.createMock(PromptServiceImpl.class);
        EasyMock.replay(promptManager);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagStaticQuery.InitConfig service = new RagStaticQuery.InitConfig();
        service.setNotifierService(notifierManager);
        service.setPromptService(promptManager);
        service.setTimeout4Condition(10086);
        RagStaticQuery empty = service.ragStaticQuery();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(promptManager, empty.getPromptService());
        Assert.assertNotNull(empty);
        EasyMock.verify(promptManager);
    }
}
