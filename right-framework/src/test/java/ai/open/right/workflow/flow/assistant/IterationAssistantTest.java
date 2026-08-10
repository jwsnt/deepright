package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.iteration.IterationService;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class IterationAssistantTest {

    @Test
    public void test() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        IterationConfig iterationConfig = new IterationConfig();
        workflowConfig.setIterationConfig(iterationConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        IterationService iterationService = EasyMock.createMock(IterationService.class);
        EasyMock.expect(iterationService.iterate(iterationConfig, workflowTask)).andReturn("HELLO WORLD");
        EasyMock.replay(iterationService);
        IterationAssistant iterationAssistant = new IterationAssistant();
        iterationAssistant.setNotifierService(notifierManager);
        iterationAssistant.setIterationService(iterationService);
        iterationAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(iterationService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        IterationService service = EasyMock.createMock(IterationService.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        IterationAssistant.InitConfig assistant = new IterationAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setIterationService(service);
        IterationAssistant empty = assistant.iterationAssistant();
        Assert.assertEquals(service, empty.getIterationService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = IterationAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = IterationAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
