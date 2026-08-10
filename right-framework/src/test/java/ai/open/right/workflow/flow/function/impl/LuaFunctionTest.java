package ai.open.right.workflow.flow.function.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionConfig;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.flow.script.impl.LuaService;
import ai.open.right.workflow.flow.script.impl.ScriptEnv;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.Collections;

public class LuaFunctionTest {

    @Test
    public void test() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setTimeout(1000);
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        LuaService luaService = EasyMock.createMock(LuaService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig.getEnvironment());
        EasyMock.expect(luaService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(luaService);
        LuaFunction luaFunction = new LuaFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        luaFunction.setLuaService(luaService);
        luaFunction.setResourceService(ObjectBuilder.buildResourceService());
        luaFunction.setTimeout(1000);
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = luaFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(luaService);
    }

    @Test
    public void testWithConfig() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript2.py");
        functionConfig.setTimeout(1000);
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        LuaService luaService = EasyMock.createMock(LuaService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig.getEnvironment());
        EasyMock.expect(luaService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript2.py")), 1000)).andReturn("WORLD").anyTimes();
        EasyMock.replay(luaService);
        LuaFunction luaFunction = new LuaFunction() {
            protected ScriptEnv buildEnv(FunctionConfig functionConfig, WorkflowTask workTask) throws Exception {
                return scriptEnv;
            }
        };
        luaFunction.setLuaService(luaService);
        luaFunction.setTimeout(1000);
        luaFunction.setResourceService(ObjectBuilder.buildResourceService());
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Object response = luaFunction.call(functionContext);
        Assert.assertEquals("WORLD", response.toString());
        EasyMock.verify(luaService);
    }

    @Test(expected = RuntimeException.class)
    public void testException() throws Exception {
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setResource("classpath:script/osScript1.py");
        functionConfig.setTimeout(1000);
        functionConfig.setEnvironment(Collections.singletonMap("ENV_KEY", "ENV_VAL"));
        LuaService luaService = EasyMock.createMock(LuaService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        scriptEnv.env(functionConfig.getEnvironment());
        EasyMock.expect(luaService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/osScript1.py")), 1000)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(luaService);
        LuaFunction luaFunction = new LuaFunction();
        luaFunction.setLuaService(luaService);
        try {
            FunctionContext functionContext = FunctionContext.builder()
                    .functionConfig(functionConfig)
                    .workTask(workflowTask)
                    .build();
            luaFunction.call(functionContext);
        } finally {
            EasyMock.verify(luaService);
        }
    }

    @Test
    public void testInit() throws Exception {
        LuaService scriptService = EasyMock.createMock(LuaService.class);
        EasyMock.replay(scriptService);
        LuaFunction.InitConfig service = new LuaFunction.InitConfig();
        service.setLuaService(scriptService);
        service.setTimeout(1000);
        LuaFunction empty = service.luaFunction();
        Assert.assertEquals(scriptService, empty.getLuaService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        EasyMock.verify(scriptService);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = LuaService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = LuaService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode3() throws Exception {
        Object object = LuaFunction.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode4() throws Exception {
        Object object = LuaFunction.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
