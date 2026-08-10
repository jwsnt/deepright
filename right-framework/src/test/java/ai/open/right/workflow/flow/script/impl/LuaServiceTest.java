package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.luaj.vm2.Globals;
import org.luaj.vm2.LuaError;
import org.luaj.vm2.lib.jse.JsePlatform;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeoutException;

public class LuaServiceTest {

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
    public void testExample1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        LuaService luaService = new LuaService();
        luaService.setResourceService(ObjectBuilder.buildResourceService());
        luaService.setTimeout(10000);
        luaService.setExecutorService(executorService);
        luaService.init();
        Assert.assertEquals("这些数的平均数是: \t145.71428\n", luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example1.lua").openStream()), 10000));
        executorService.shutdown();
    }

    @Test(expected = WorkflowException.class)
    public void testExample2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        LuaService luaService = new LuaService();
        luaService.setResourceService(ObjectBuilder.buildResourceService());
        luaService.setTimeout(10000);
        luaService.setExecutorService(executorService);
        luaService.init();
        try {
            luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example2.lua").openStream()), 10000);
        } catch (WorkflowException e) {
            Assert.assertEquals("\n" +
                    "error(\"平均数大于 50，触发异常！\")\n" +
                    ":2 平均数大于 50，触发异常！", e.getMessage());
            throw e;
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testExample3() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        LuaService luaService = new LuaService();
        luaService.setResourceService(ObjectBuilder.buildResourceService());
        luaService.setTimeout(10000);
        luaService.setExecutorService(executorService);
        luaService.init();
        Assert.assertEquals("", luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example3.lua").openStream()), 10000));
        executorService.shutdown();
    }

    @Test(expected = TimeoutException.class)
    public void testExample4() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        LuaService luaService = new LuaService();
        luaService.setResourceService(ObjectBuilder.buildResourceService());
        luaService.setTimeout(5000);
        luaService.setExecutorService(executorService);
        luaService.init();
        try {
            Assert.assertEquals("", luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example4.lua").openStream()), 10000));
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testExampleMulti() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        LuaService luaService = new LuaService();
        luaService.setResourceService(ObjectBuilder.buildResourceService());
        luaService.setTimeout(10000);
        luaService.setExecutorService(executorService);
        luaService.init();
        try {
            luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example2.lua").openStream()), 10000);
        } catch (WorkflowException e) {
            Assert.assertEquals("\n" +
                    "error(\"平均数大于 50，触发异常！\")\n" +
                    ":2 平均数大于 50，触发异常！", e.getMessage());
        }
        Assert.assertNotNull(luaService.getResourceService());
        Assert.assertEquals("这些数的平均数是: \t145.71428\n", luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example1.lua").openStream()), 10000));
        try {
            luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example2.lua").openStream()), 10000);
        } catch (WorkflowException e) {
            Assert.assertEquals("\n" +
                    "error(\"平均数大于 50，触发异常！\")\n" +
                    ":2 平均数大于 50，触发异常！", e.getMessage());
        }
        try {
            Assert.assertEquals("", luaService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example4.lua").openStream()), 10000));
            Assert.fail();
        } catch (Exception e) {
        } finally {
            executorService.shutdown();
        }
    }

    @Test(expected = LuaError.class)
    public void testRelease() throws Exception {
        GenericObjectPool pool = EasyMock.createMock(GenericObjectPool.class);
        Globals globals = JsePlatform.standardGlobals();
        EasyMock.expect(pool.borrowObject(1000)).andReturn(globals).anyTimes();
        pool.returnObject(globals);
        EasyMock.expectLastCall().anyTimes();
        pool.invalidateObject(globals);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(pool);
        try {
            LuaService.LuaFuture luaFuture = new LuaService.LuaFuture(pool, new ScriptEnv(ObjectBuilder.buildWorkflowTask()), 1000, "JS", IOUtils.toString(ResourceUtils.getURL("classpath:dkjson.lua").openStream(), StandardCharsets.UTF_8));
            luaFuture.call();
        } finally {
            EasyMock.verify(pool);
        }
    }

    @Test
    public void testExampleWithEnv() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        LuaService luaService = new LuaService();
        luaService.setResourceService(ObjectBuilder.buildResourceService());
        luaService.setTimeout(5000);
        luaService.setExecutorService(executorService);
        luaService.init();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.getMetadata().put("HELLO", "WORLD");
        ScriptEnv scriptEnv = new ScriptEnv(workflowTask);
        // {__workflow__={"workflow":"UNKNOWN","biz":"UNKNOWN"}, __user__={"language":"UNKNOWN","system":"UNKNOWN","device":"UNKNOWN","region":"UNKNOWN","brand":"UNKNOWN","model":"UNKNOWN","token":"UNKNOWN"}, __metadata__={"HELLO":"WORLD"}}
        Assert.assertEquals("Workflow: UNKNOWN\n" +
                "Biz: UNKNOWN\n" +
                "Metadata: WORLD\n" +
                "User: UNKNOWN\n", luaService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/example5.lua").openStream()), 10000));
        executorService.shutdown();
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        EasyMock.replay(executorService);
        LuaService.InitConfig service = new LuaService.InitConfig();
        service.setNotifierService(notifierManager);
        service.setResourceService(ObjectBuilder.buildResourceService());
        service.setTimeout(100);
        service.setExecutorService(executorService);
        service.setTimeout4Corrector(200);
        service.setTimeout4Condition(100);
        LuaService empty = service.luaService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout4Condition());
        EasyMock.verify(executorService);
        Assert.assertNull(empty.getObjectPool());
        empty.init();
        Assert.assertNotNull(empty.getObjectPool());
        Assert.assertNotNull(empty.getDkjson());
        empty.destroy();
    }
    @Test
    public void testCleanNoTags() {
        LuaService service = new LuaService();
        Assert.assertEquals("print(1)", service.clean("  print(1)  "));
    }

    @Test
    public void testPrintFunctionMultipleArgs() {
        LuaService.PrintFunction pf = new LuaService.PrintFunction();
        pf.invoke(org.luaj.vm2.LuaValue.varargsOf(new org.luaj.vm2.LuaValue[]{
            org.luaj.vm2.LuaValue.valueOf("A"), org.luaj.vm2.LuaValue.valueOf("B")
        }));
        Assert.assertEquals("A\tB\n", pf.buildContent());
    }
}
