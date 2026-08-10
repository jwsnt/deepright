package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class PassageAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setDiscard(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PassageAssistant passageAssistant = new PassageAssistant();
        passageAssistant.execute(workflowConfig, workflowTask);
    }

    @Test
    public void testWithChain() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setDiscard(false);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        PassageAssistant passageAssistant = new PassageAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content, Integer code) throws Exception {
            }
        };
        passageAssistant.execute(workflowConfig, workflowTask);
    }

    @Test
    public void testClose1() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setClose(true);
        workflowConfig.setDiscard(false);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger count = new AtomicInteger(0);
        PassageAssistant passageAssistant = new PassageAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content, Integer code) throws Exception {
                count.decrementAndGet();
            }

            @Override
            protected void close(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
                count.incrementAndGet();
            }
        };
        passageAssistant.execute(workflowConfig, workflowTask);
        Assert.assertEquals(1, count.get());
    }

    @Test
    public void testClose2() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setClose(true);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger count = new AtomicInteger(0);
        PassageAssistant passageAssistant = new PassageAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content, Integer code) throws Exception {
                count.incrementAndGet();
            }

            @Override
            protected void close(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
                count.incrementAndGet();
            }
        };
        passageAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        passageAssistant.execute(workflowConfig, workflowTask);
        Assert.assertEquals(1, count.get());
    }

    @Test
    public void testClose3() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setCode(500);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger count = new AtomicInteger(0);
        PassageAssistant passageAssistant = new PassageAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content, Integer code) throws Exception {
                count.incrementAndGet();
                Assert.assertEquals(Integer.valueOf(500), code);
            }

            @Override
            protected void close(WorkflowConfig config, WorkflowTask workTask) throws Exception {
                count.incrementAndGet();
            }
        };
        passageAssistant.execute(workflowConfig, workflowTask);
        Assert.assertEquals(1, count.get());
    }


    @Test
    public void testCloseFunction() throws Exception {
        PassageAssistant passageAssistant = new PassageAssistant();
        passageAssistant.setNotifierService(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals(Integer.valueOf(0), segment.getCode());
                Assert.assertEquals(Notifier.SOURCE, segment.getNotifier());
            }
        });
        WorkflowConfig workflowConfig = new WorkflowConfig();
        passageAssistant.close(workflowConfig, ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        EasyMock.replay(notifierManager, signalFactory);
        PassageAssistant.InitConfig assistant = new PassageAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        PassageAssistant empty = assistant.passageAssistant();
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = PassageAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PassageAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
