package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagOrchestrator;
import ai.open.right.workflow.flow.llm.rag.digest.RagDigest;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class RagFlowTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagFlow.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagFlow.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testNotAllow() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("true");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        RagFlow flow = new RagFlow();
        flow.setNotifierService(notifierManager);
        RagFuture ragFuture = flow.rag(ragConfig, ragData);
        ragFuture.run();
    }

    @Test
    public void testNotAllowWithOutBefore() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("true");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        RagOrchestrator ragOrchestrator = new RagOrchestrator();
        ragOrchestrator.setAfter("AFTER");
        ragConfig.setRagOrchestrator(ragOrchestrator);
        RagFlow flow = new RagFlow();
        flow.setNotifierService(notifierManager);
        RagFuture ragFuture = flow.rag(ragConfig, ragData);
        ragFuture.run();
    }

    @Test
    public void testNotAllowWithBefore() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("true");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        RagOrchestrator ragOrchestrator = new RagOrchestrator();
        ragOrchestrator.setBefore("After");
        ragConfig.setRagOrchestrator(ragOrchestrator);
        RagFlow flow = new RagFlow();
        flow.setNotifierService(notifierManager);
        RagFuture ragFuture = flow.rag(ragConfig, ragData);
        ragFuture.run();
    }

    @Test
    public void testNotAllowWithBeforeAndAfter() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("true");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        RagOrchestrator ragOrchestrator = new RagOrchestrator();
        ragOrchestrator.setAfter("After");
        ragOrchestrator.setBefore("Before");
        ragConfig.setRagOrchestrator(ragOrchestrator);
        RagFlow flow = new RagFlow();
        flow.setNotifierService(notifierManager);
        RagFuture ragFuture = flow.rag(ragConfig, ragData);
        ragFuture.run();
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagFlow.InitConfig service = new RagFlow.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Llm(1000);
        service.setTimeout4Condition(10086);
        RagFlow empty = service.ragFlow();
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
    }
}
