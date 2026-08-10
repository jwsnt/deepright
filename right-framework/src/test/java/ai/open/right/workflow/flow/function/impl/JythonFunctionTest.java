package ai.open.right.workflow.flow.function.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.function.impl.JythonFunction;
import ai.open.right.workflow.flow.script.impl.JavaScriptService;
import ai.open.right.workflow.flow.script.impl.JythonService;
import ai.open.right.workflow.flow.script.impl.ScriptEnv;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.Collections;

public class JythonFunctionTest {

    @Test
    public void test() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        JythonService jythonService = EasyMock.createMock(JythonService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig);
        EasyMock.expect(jythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(jythonService);
        JythonFunction jythonFunction = new JythonFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        jythonFunction.setJythonService(jythonService);
        jythonFunction.setTimeout(1000);
        jythonFunction.setResourceService(ObjectBuilder.buildResourceService());
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = jythonFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(jythonService);
    }

    @Test
    public void testWithConfig() throws Exception {
        JythonService jythonService = EasyMock.createMock(JythonService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        scriptEnv.put("ONE", "TWO");
        EasyMock.expect(jythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript2.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(jythonService);
        JythonFunction jythonFunction = new JythonFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        jythonFunction.setJythonService(jythonService);
        jythonFunction.setResourceService(ObjectBuilder.buildResourceService());
        jythonFunction.setTimeout(1000);
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript2.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = jythonFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(jythonService);
    }

    @Test(expected = RuntimeException.class)
    public void testException() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setTimeout(1000);
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig.getEnvironment());
        JythonService jythonService = EasyMock.createMock(JythonService.class);
        EasyMock.expect(jythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(jythonService);
        JythonFunction jythonFunction = new JythonFunction();
        jythonFunction.setJythonService(jythonService);
        try {
            FunctionContext functionContext = FunctionContext.builder()
                    .functionConfig(functionConfig)
                    .workTask(workflowTask)
                    .build();
            jythonFunction.call(functionContext);
        } finally {
            EasyMock.verify(jythonService);
        }
    }

    @Test
    public void testInit() throws Exception {
        JythonService scriptService = EasyMock.createMock(JythonService.class);
        EasyMock.replay(scriptService);
        JythonFunction.InitConfig service = new JythonFunction.InitConfig();
        service.setJythonService(scriptService);
        service.setTimeout(1000);
        JythonFunction empty = service.jythonFunction();
        Assert.assertEquals(scriptService, empty.getJythonService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        EasyMock.verify(scriptService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = JythonFunction.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = JythonFunction.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
