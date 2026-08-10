package ai.open.right.workflow.flow.function.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.JythonService;
import ai.open.right.workflow.flow.script.impl.PolyglotService;
import ai.open.right.workflow.flow.script.impl.ScriptEnv;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.Collections;

public class PolyglotFunctionTest {

    @Test
    public void test() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        PolyglotService polyglotService = EasyMock.createMock(PolyglotService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig);
        EasyMock.expect(polyglotService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(polyglotService);
        PolyglotFunction polyglotFunction = new PolyglotFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        polyglotFunction.setPolyglotService(polyglotService);
        polyglotFunction.setTimeout(1000);
        polyglotFunction.setResourceService(ObjectBuilder.buildResourceService());
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = polyglotFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(polyglotService);
    }

    @Test
    public void testWithConfig() throws Exception {
        PolyglotService polyglotService = EasyMock.createMock(PolyglotService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.put("HELLO", "UNKNOWN");
        scriptEnv.put("ONE", "TWO");
        EasyMock.expect(polyglotService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript2.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(polyglotService);
        PolyglotFunction polyglotFunction = new PolyglotFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        polyglotFunction.setPolyglotService(polyglotService);
        polyglotFunction.setTimeout(1000);
        polyglotFunction.setResourceService(ObjectBuilder.buildResourceService());
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript2.py");
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = polyglotFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(polyglotService);
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
        PolyglotService polyglotService = EasyMock.createMock(PolyglotService.class);
        EasyMock.expect(polyglotService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(polyglotService);
        PolyglotFunction polyglotFunction = new PolyglotFunction();
        polyglotFunction.setPolyglotService(polyglotService);
        try {
            FunctionContext functionContext = FunctionContext.builder()
                    .functionConfig(functionConfig)
                    .workTask(workflowTask)
                    .build();
            polyglotFunction.call(functionContext);
        } finally {
            EasyMock.verify(polyglotService);
        }
    }

    @Test
    public void testInit() throws Exception {
        PolyglotService scriptService = EasyMock.createMock(PolyglotService.class);
        EasyMock.replay(scriptService);
        PolyglotFunction.InitConfig service = new PolyglotFunction.InitConfig();
        service.setPolyglotService(scriptService);
        service.setTimeout(1000);
        PolyglotFunction empty = service.polyglotFunction();
        Assert.assertEquals(scriptService, empty.getPolyglotService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        EasyMock.verify(scriptService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = PolyglotFunction.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PolyglotFunction.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
