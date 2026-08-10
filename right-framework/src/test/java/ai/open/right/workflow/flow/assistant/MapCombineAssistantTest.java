package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.mapcombine.MapCombineConfig;
import ai.open.right.workflow.flow.mapcombine.MapCombineService;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class MapCombineAssistantTest {

    @Test
    public void testExecuteWithoutChain() throws Exception {
        MapCombineService mapCombineService = EasyMock.createMock(MapCombineService.class);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        workflowConfig.setMapCombineConfig(mapCombineConfig);
        EasyMock.expect(mapCombineService.execute(mapCombineConfig, workTask)).andReturn("Hello World").anyTimes();
        EasyMock.replay(mapCombineService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        MapCombineAssistant mapCombineAssistant = new MapCombineAssistant();
        mapCombineAssistant.setMapCombineService(mapCombineService);
        mapCombineAssistant.setNotifierService(notifierManager);
        mapCombineAssistant.execute(workflowConfig, workTask);
        EasyMock.verify(mapCombineService);
    }

    @Test
    public void testExecuteWithChain() throws Exception {
        MapCombineService mapCombineService = EasyMock.createMock(MapCombineService.class);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("NextWorkflow");
        MapCombineConfig mapCombineConfig = new MapCombineConfig();
        workflowConfig.setMapCombineConfig(mapCombineConfig);
        EasyMock.expect(mapCombineService.execute(mapCombineConfig, workTask)).andReturn("Hello World").anyTimes();
        EasyMock.replay(mapCombineService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        MapCombineAssistant mapCombineAssistant = new MapCombineAssistant();
        mapCombineAssistant.setMapCombineService(mapCombineService);
        mapCombineAssistant.setNotifierService(notifierManager);
        mapCombineAssistant.execute(workflowConfig, workTask);
        EasyMock.verify(mapCombineService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        MapCombineService service = EasyMock.createMock(MapCombineService.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        MapCombineAssistant.InitConfig assistant = new MapCombineAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setMapCombineService(service);
        MapCombineAssistant empty = assistant.mapCombineAssistant();
        Assert.assertEquals(service, empty.getMapCombineService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = MapCombineAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = MapCombineAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
