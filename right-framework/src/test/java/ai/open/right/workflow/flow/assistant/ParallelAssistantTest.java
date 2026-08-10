package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.parallel.ParallelConfig;
import ai.open.right.workflow.flow.parallel.ParallelService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ParallelAssistantTest {

    @Test
    public void testExecuteWithoutChain() throws Exception {
        ParallelService parallelService = EasyMock.createMock(ParallelService.class);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ParallelConfig parallelConfig = new ParallelConfig();
        workflowConfig.setParallelConfig(parallelConfig);
        EasyMock.expect(parallelService.execute(parallelConfig, workTask)).andReturn("Hello World").anyTimes();
        EasyMock.replay(parallelService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ParallelAssistant parallelAssistant = new ParallelAssistant();
        parallelAssistant.setParallelService(parallelService);
        parallelAssistant.setNotifierService(notifierManager);
        parallelAssistant.execute(workflowConfig, workTask);
        EasyMock.verify(parallelService);
    }

    @Test
    public void testExecuteWithChain() throws Exception {
        ParallelService parallelService = EasyMock.createMock(ParallelService.class);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("NextWorkflow");
        ParallelConfig parallelConfig = new ParallelConfig();
        workflowConfig.setParallelConfig(parallelConfig);
        EasyMock.expect(parallelService.execute(parallelConfig, workTask)).andReturn("Hello World").anyTimes();
        EasyMock.replay(parallelService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ParallelAssistant parallelAssistant = new ParallelAssistant();
        parallelAssistant.setParallelService(parallelService);
        parallelAssistant.setNotifierService(notifierManager);
        parallelAssistant.execute(workflowConfig, workTask);
        EasyMock.verify(parallelService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        ParallelService service = EasyMock.createMock(ParallelService.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        ParallelAssistant.InitConfig assistant = new ParallelAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setParallelService(service);
        ParallelAssistant empty = assistant.parallelAssistant();
        Assert.assertEquals(service, empty.getParallelService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = ParallelAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ParallelAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
