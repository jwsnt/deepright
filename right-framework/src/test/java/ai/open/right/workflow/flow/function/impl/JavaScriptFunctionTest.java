package ai.open.right.workflow.flow.function.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.mcp.McpPromptGetAssistant;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.JavaScriptService;
import ai.open.right.workflow.flow.script.impl.ScriptEnv;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.Collections;

public class JavaScriptFunctionTest {

    @Test
    public void test() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        EasyMock.expect(javaScriptService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(javaScriptService);
        JavaScriptFunction javaScriptFunction = new JavaScriptFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        javaScriptFunction.setJavaScriptService(javaScriptService);
        javaScriptFunction.setTimeout(1000);
        javaScriptFunction.setResourceService(ObjectBuilder.buildResourceService());
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = javaScriptFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(javaScriptService);
    }

    @Test
    public void testWithConfig() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        scriptEnv.put("ONE", "TWO");
        EasyMock.expect(javaScriptService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript2.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(javaScriptService);
        JavaScriptFunction javaScriptFunction = new JavaScriptFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        javaScriptFunction.setJavaScriptService(javaScriptService);
        javaScriptFunction.setTimeout(1000);
        javaScriptFunction.setResourceService(ObjectBuilder.buildResourceService());
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript2.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = javaScriptFunction.call(functionContext);
        Assert.assertNotNull(javaScriptFunction.getResourceService());
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(javaScriptService);
    }

    @Test(expected = RuntimeException.class)
    public void testException() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        EasyMock.expect(javaScriptService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(javaScriptService);
        JavaScriptFunction javaScriptFunction = new JavaScriptFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        javaScriptFunction.setJavaScriptService(javaScriptService);
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setTimeout(1000);
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        try {
            FunctionContext functionContext = FunctionContext.builder()
                    .functionConfig(functionConfig)
                    .workTask(workflowTask)
                    .build();
            javaScriptFunction.call(functionContext);
        } finally {
            EasyMock.verify(javaScriptService);
        }
    }

    @Test
    public void testInit() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        EasyMock.replay(javaScriptService);
        JavaScriptFunction.InitConfig service = new JavaScriptFunction.InitConfig();
        service.setResourceService(ObjectBuilder.buildResourceService());
        service.setJavaScriptService(javaScriptService);
        service.setTimeout(1000);
        JavaScriptFunction empty = (JavaScriptFunction) service.javaScriptFunction();
        Assert.assertNotNull(empty.getResourceService());
        Assert.assertEquals(javaScriptService, empty.getJavaScriptService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        EasyMock.verify(javaScriptService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = JavaScriptService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = JavaScriptService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode3() throws Exception {
        Object object = JavaScriptFunction.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode4() throws Exception {
        Object object = JavaScriptFunction.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
