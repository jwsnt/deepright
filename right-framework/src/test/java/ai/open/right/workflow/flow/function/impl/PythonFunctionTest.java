package ai.open.right.workflow.flow.function.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.PythonService;
import ai.open.right.workflow.flow.script.impl.ScriptEnv;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.Collections;

public class PythonFunctionTest {

    @Test
    public void test() throws Exception {
        PythonService pythonService = EasyMock.createMock(PythonService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        EasyMock.expect(pythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(pythonService);
        PythonFunction pythonFunction = new PythonFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        pythonFunction.setPythonService(pythonService);
        pythonFunction.setResourceService(ObjectBuilder.buildResourceService());
        pythonFunction.setTimeout(1000);
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = pythonFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(pythonService);
    }

    @Test
    public void testWithConfig() throws Exception {
        PythonService pythonService = EasyMock.createMock(PythonService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        scriptEnv.put("ONE", "TWO");
        EasyMock.expect(pythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript2.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(pythonService);
        PythonFunction pythonFunction = new PythonFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        pythonFunction.setPythonService(pythonService);
        pythonFunction.setResourceService(ObjectBuilder.buildResourceService());
        pythonFunction.setTimeout(1000);
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript2.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = pythonFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(pythonService);
    }

    @Test(expected = RuntimeException.class)
    public void testException() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setTimeout(1000);
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        PythonService pythonService = EasyMock.createMock(PythonService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig);
        EasyMock.expect(pythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(pythonService);
        PythonFunction pythonFunction = new PythonFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        pythonFunction.setPythonService(pythonService);
        try {
            FunctionContext functionContext = FunctionContext.builder()
                    .functionConfig(functionConfig)
                    .workTask(workflowTask)
                    .build();
            pythonFunction.call(functionContext);
        } finally {
            EasyMock.verify(pythonService);
        }
    }

    @Test
    public void testInit() throws Exception {
        PythonService scriptService = EasyMock.createMock(PythonService.class);
        EasyMock.replay(scriptService);
        PythonFunction.InitConfig service = new PythonFunction.InitConfig();
        service.setPythonService(scriptService);
        service.setTimeout(1000);
        PythonFunction empty = service.pythonFunction();
        Assert.assertEquals(scriptService, empty.getPythonService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        EasyMock.verify(scriptService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = PythonFunction.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PythonFunction.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
    @Test(expected = IllegalArgumentException.class)
    public void testCallNoResource() throws Exception {
        PythonFunction function = new PythonFunction();
        FunctionConfig config = new FunctionConfig();
        config.setResource(null);
        FunctionContext context = FunctionContext.builder().functionConfig(config).build();
        function.call(context);
    }
}
