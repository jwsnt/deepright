package ai.open.right.workflow.flow.function.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.sync.SyncConfig;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class BaseFunctionTest {

    @Test
    public void testSource1() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Map<String, Object> _meta = new HashMap<>();
        Integer _code = 123;
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
                Assert.assertEquals(Integer.valueOf(_code), segment.getCode());
                Assert.assertEquals(_meta, segment.getMetadata());
            }
        });
        baseFunction.source(_functionContext, _meta, "OK", _code);
    }

    @Test
    public void testSource2() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Map<String, Object> _meta = new HashMap<>();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
                Assert.assertEquals(_meta, segment.getMetadata());
            }
        });
        baseFunction.source(_functionContext, _meta, "OK");
    }

    @Test
    public void testSource3() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
            }
        });
        baseFunction.source(_functionContext, "OK");
    }

    @Test
    public void testSource4() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(Integer.valueOf(250), segment.getCode());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
            }
        });
        baseFunction.source(_functionContext, "OK", 250);
    }

    @Test
    public void testEndpoint1() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Map<String, Object> _meta = new HashMap<>();
        Integer _code = 123;
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
                Assert.assertEquals(Integer.valueOf(_code), segment.getCode());
                Assert.assertEquals(_meta, segment.getMetadata());
            }
        });
        baseFunction.endpoint(_functionContext, _meta, "OK", _code);
    }

    @Test
    public void testEndpoint2() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Map<String, Object> _meta = new HashMap<>();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
                Assert.assertEquals(_meta, segment.getMetadata());
            }
        });
        baseFunction.endpoint(_functionContext, _meta, "OK");
    }

    @Test
    public void testEndpoint3() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
            }
        });
        baseFunction.endpoint(_functionContext, "OK");
    }

    @Test
    public void testLocalhost() throws Exception {
        FunctionContext _functionContext = FunctionContext.builder()
                .workTask(ObjectBuilder.buildWorkflowTask())
                .build();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        baseFunction.localhost(_functionContext, SyncConfig.builder().build());
    }

    @Test
    public void testEndpoint4() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Integer _code = 501;
        FunctionContext _functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        BaseFunction baseFunction = new BaseFunction() {
            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(_functionContext, functionContext);
                return null;
            }
        };
        baseFunction.setNotifierService(new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("OK", segment.getContent());
                Assert.assertEquals(redirectContext, workflowTask);
                Assert.assertEquals(notifierWriteBack, workflowTask);
                Assert.assertEquals(_code, segment.getCode());
            }
        });
        baseFunction.endpoint(_functionContext, "OK", _code);
    }

    @Test
    public void testCall() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        BaseFunction baseFunction = new BaseFunction();
        Assert.assertEquals(workflowTask.getQuery(), baseFunction.call(functionContext));
    }
    @Test
    public void testCallDefault() throws Exception {
        BaseFunction function = new BaseFunction();
        WorkflowTask task = ObjectBuilder.buildWorkflowTask();
        task.setQuery("QUERY");
        FunctionContext context = FunctionContext.builder().workTask(task).build();
        Assert.assertEquals("QUERY", function.call(context));
    }
}
