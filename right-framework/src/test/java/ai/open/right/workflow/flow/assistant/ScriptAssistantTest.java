package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.script.ScriptConfig;
import ai.open.right.workflow.flow.script.ScriptSegment;
import ai.open.right.workflow.flow.script.impl.ScriptServiceImpl;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.notify.NotifierWriteBack;
import com.fasterxml.jackson.core.JsonParseException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ScriptAssistantTest {

    @Test(expected = IllegalArgumentException.class)
    public void testWithoutQuery() throws Exception {
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("");
        scriptAssistant.execute(new WorkflowConfig(), workflowTask);
    }

    @Test
    public void testWithoutChain() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        workflowConfig.setScriptConfig(scriptConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andReturn("World").anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        scriptAssistant.setNotifierService(notifierManager);
        scriptAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(scriptService);
    }

    @Test
    public void testWithChain() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andReturn("World").anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        scriptAssistant.setNotifierService(notifierManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setScriptConfig(scriptConfig);
        workflowConfig.setChain("NextWorkflow");
        scriptAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(scriptService);
    }

    @Test(expected = WorkflowException.class)
    public void testWithOutCode() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        workflowConfig.setScriptConfig(scriptConfig);
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andThrow(new WorkflowException("HELLO", 201)).anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        try {
            scriptAssistant.execute(workflowConfig, workflowTask);
        } finally {
            EasyMock.verify(scriptService);
        }
    }

    @Test(expected = WorkflowException.class)
    public void testWithNotCode200() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        workflowConfig.setScriptConfig(scriptConfig);
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andThrow(new WorkflowException("HELLO", 201)).anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        try {
            scriptAssistant.execute(workflowConfig, workflowTask);
        } finally {
            EasyMock.verify(scriptService);
        }
    }


    @Test
    public void testWithDef() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        workflowConfig.setScriptConfig(scriptConfig);
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        scriptAssistant.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals("HELLO", segment.getContent());
            }
        });
        scriptAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(scriptService);
    }

    @Test(expected = JsonParseException.class)
    public void testWithWrapAndException() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        workflowConfig.setScriptConfig(scriptConfig);
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        scriptAssistant.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.fail();
            }
        });
        try {
            scriptAssistant.execute(workflowConfig, workflowTask);
        } finally {
            EasyMock.verify(scriptService);
        }
    }

    @Test
    public void testWithWrapAndErrorCode() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        workflowConfig.setScriptConfig(scriptConfig);
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andReturn("{\"code\":201,\"data\":\"HELLO\"}").anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        scriptAssistant.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("HELLO", segment.getContent());
                Assert.assertEquals(ScriptSegment.class.cast(segment).getData(), "HELLO");
            }
        });
        scriptAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(scriptService);
    }

    @Test
    public void testWithWrapAndRightCode() throws Exception {
        ScriptServiceImpl scriptService = EasyMock.createMock(ScriptServiceImpl.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        workflowConfig.setScriptConfig(scriptConfig);
        EasyMock.expect(scriptService.run(scriptConfig, workflowTask)).andReturn("{\"code\":200,\"data\":\"HELLO\"}").anyTimes();
        EasyMock.replay(scriptService);
        ScriptAssistant scriptAssistant = new ScriptAssistant();
        scriptAssistant.setScriptService(scriptService);
        scriptAssistant.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals("HELLO", segment.getContent());
            }
        });
        scriptAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(scriptService);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        ScriptServiceImpl service = EasyMock.createMock(ScriptServiceImpl.class);
        EasyMock.replay(service, notifierManager, signalFactory);
        ScriptAssistant.InitConfig assistant = new ScriptAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setScriptService(service);
        ScriptAssistant empty = assistant.scriptAssistant();
        Assert.assertEquals(service, empty.getScriptService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = ScriptAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ScriptAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
