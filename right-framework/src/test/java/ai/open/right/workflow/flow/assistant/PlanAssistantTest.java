package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.iteration.IterationConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.plan.PlanConfig;
import ai.open.right.workflow.flow.plan.PlanService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class PlanAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        PlanConfig planConfig = new PlanConfig();
        planConfig.setIterationConfig(new IterationConfig());
        _workflowConfig.setPlanConfig(planConfig);
        WorkflowTask _workTask = ObjectBuilder.buildWorkflowTask();
        PlanService planService = EasyMock.createMock(PlanService.class);
        EasyMock.expect(planService.plan(planConfig, _workTask)).andReturn("OK").anyTimes();
        EasyMock.replay(planService);
        PlanAssistant planAssistant = new PlanAssistant();
        planAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        planAssistant.setPlanService(planService);
        planAssistant.execute(_workflowConfig, _workTask);
        EasyMock.verify(planService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        PlanService service = EasyMock.createMock(PlanService.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        PlanAssistant.InitConfig assistant = new PlanAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setPlanService(service);
        PlanAssistant empty = assistant.planAssistant();
        Assert.assertEquals(service, empty.getPlanService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = PlanAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PlanAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
