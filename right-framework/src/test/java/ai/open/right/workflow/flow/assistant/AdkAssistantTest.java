package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.adk.AdkService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class AdkAssistantTest {

    @Test
    public void testAdkAssistant() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        AdkService adkService = EasyMock.createMock(AdkService.class);
        EasyMock.expect(adkService.execute(workflowConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(adkService);
        AdkAssistant assistant = new AdkAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("HELLO", content);
            }
        };
        assistant.setAdkService(adkService);
        assistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(adkService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        AdkService adkService = EasyMock.createMock(AdkService.class);
        EasyMock.replay(adkService, notifierManager, signalFactory);
        AdkAssistant.InitConfig initConfig = new AdkAssistant.InitConfig();
        initConfig.setNotifierService(notifierManager);
        initConfig.setLlmQueryService(llmQueryServices);
        initConfig.setSignalFactory(signalFactory);
        initConfig.setAdkService(adkService);
        AdkAssistant assistant = initConfig.adkAssistant();
        Assert.assertEquals(assistant.getAdkService(), initConfig.getAdkService());
        Assert.assertEquals(assistant.getLlmQueryService(), initConfig.getLlmQueryService());
        Assert.assertEquals(assistant.getSignalFactory(), initConfig.getSignalFactory());
        Assert.assertEquals(assistant.getNotifierService(), initConfig.getNotifierService());
        EasyMock.verify(adkService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = AdkAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = AdkAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
