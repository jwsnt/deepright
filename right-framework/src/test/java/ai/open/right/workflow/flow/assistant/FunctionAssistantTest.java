package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionResponse;
import ai.open.right.workflow.flow.function.impl.FunctionServiceImpl;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class FunctionAssistantTest {

    @Test
    public void test() throws Exception {
        FunctionServiceImpl functionManager = EasyMock.createMock(FunctionServiceImpl.class);
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(functionManager.call(functionConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(functionManager);
        FunctionAssistant functionAssistant = new FunctionAssistant();
        functionAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        functionAssistant.setFunctionService(functionManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setFunctionConfig(functionConfig);
        functionAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(functionManager);
    }

    @Test(expected = WorkflowException.class)
    public void testAndException() throws Exception {
        FunctionServiceImpl functionManager = EasyMock.createMock(FunctionServiceImpl.class);
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(functionManager.call(functionConfig, workflowTask)).andThrow(new WorkflowException()).anyTimes();
        EasyMock.replay(functionManager);
        FunctionAssistant functionAssistant = new FunctionAssistant();
        functionAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent());
        functionAssistant.setFunctionService(functionManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setFunctionConfig(functionConfig);
        functionAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(functionManager);
    }

    @Test(expected = WorkflowException.class)
    public void testAndExceptionWithWorkflow() throws Exception {
        FunctionServiceImpl functionManager = EasyMock.createMock(FunctionServiceImpl.class);
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(functionManager.call(functionConfig, workflowTask)).andThrow(new WorkflowException()).anyTimes();
        EasyMock.replay(functionManager);
        FunctionAssistant functionAssistant = new FunctionAssistant();
        functionAssistant.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                throw new RuntimeException();
            }
        });
        functionAssistant.setFunctionService(functionManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setFunctionConfig(functionConfig);
        functionAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(functionManager);
    }

    @Test(expected = RuntimeException.class)
    public void testAndExceptionWithRuntime() throws Exception {
        FunctionServiceImpl functionManager = EasyMock.createMock(FunctionServiceImpl.class);
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(functionManager.call(functionConfig, workflowTask)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(functionManager);
        FunctionAssistant functionAssistant = new FunctionAssistant();
        functionAssistant.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                throw new RuntimeException();
            }
        });
        functionAssistant.setFunctionService(functionManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setFunctionConfig(functionConfig);
        functionAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(functionManager);
    }

    @Test
    public void testWithFunctionResponse() throws Exception {
        FunctionServiceImpl functionManager = EasyMock.createMock(FunctionServiceImpl.class);
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        _workflowConfig.setFunctionConfig(functionConfig);
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> _metadata = new HashMap<>();
        FunctionResponse functionResponse = FunctionResponse.builder()
                .content("HELLO WORLD")
                .metadata(_metadata)
                .code(123)
                .build();
        EasyMock.expect(functionManager.call(functionConfig, _workflowTask)).andReturn(functionResponse).anyTimes();
        EasyMock.replay(functionManager);
        FunctionAssistant functionAssistant = new FunctionAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(metadata, _metadata);
                Assert.assertEquals("HELLO WORLD", content);
                Assert.assertEquals(Integer.valueOf(123), code);
            }
        };
        functionAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        functionAssistant.setFunctionService(functionManager);
        functionAssistant.execute(_workflowConfig, _workflowTask);
        EasyMock.verify(functionManager);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        FunctionServiceImpl service = EasyMock.createMock(FunctionServiceImpl.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        FunctionAssistant.InitConfig assistant = new FunctionAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setFunctionService(service);
        FunctionAssistant empty = assistant.functionAssistant();
        Assert.assertEquals(service, empty.getFunctionService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = FunctionAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = FunctionAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
