package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.easymock.EasyMock;
import org.graalvm.polyglot.PolyglotException;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeoutException;

public class PolyglotServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = PolyglotService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PolyglotService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testShortScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.setTimeout(5000);
        polyglotService.init();
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "print(1+1)", 5000), "2\n");
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test
    public void testLongScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/longScript.py").openStream(), "UTF-8");
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 15000), "Before sleep\n" + "After sleep\n");
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test(expected = TimeoutException.class)
    public void testTimeoutScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/timeoutScript.py").openStream(), "UTF-8");
        try {
            polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 10000);
            Assert.fail();
        } finally {
            executorService.shutdown();
            polyglotService.destroy();
        }
    }

    @Test(expected = ExecutionException.class)
    public void testErrorScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/noModuleScript.py").openStream(), "UTF-8");
        try {
            polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 10000);
            Assert.fail();
        } finally {
            executorService.shutdown();
            polyglotService.destroy();
        }
    }

    @Test(expected = ExecutionException.class)
    public void testRunFailedScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/runningFailScript.py").openStream(), "UTF-8");
        try {
            polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000);
            Assert.fail();
        } finally {
            executorService.shutdown();
            polyglotService.destroy();
        }
    }

    @Test(expected = ExecutionException.class)
    public void testLongTimeErrorSleep() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/sleepAndError.py").openStream(), "UTF-8");
        polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000);
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test
    public void testExtract() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "hello```python\r\nprint(1+1)\r\n```world", 5000), "2\n");
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test
    public void testClean() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "```python\r\nprint(1+1)\r\n```", 5000), "2\n");
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test
    public void testCheckJson1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson1.py").openStream(), "UTF-8");
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}\n", polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test
    public void testCheckJson2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson2.py").openStream(), "UTF-8");
        try {
            Assert.assertEquals("错误的手机号码\n", polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
        } finally {
            executorService.shutdown();
            polyglotService.destroy();
        }
    }

    @Test
    public void testMulti() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService polyglotService = new PolyglotService();
        polyglotService.setExecutorService(executorService);
        polyglotService.setEmbedding(false);
        polyglotService.init();
        String script3 = IOUtils.toString(ResourceUtils.getURL("classpath:script/timeoutScript.py").openStream(), "UTF-8");
        try {
            polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script3, 10000);
            Assert.fail();
        } catch (Exception e) {
            Assert.assertEquals(TimeoutException.class, e.getClass());
        }
        String script4 = IOUtils.toString(ResourceUtils.getURL("classpath:script/noModuleScript.py").openStream(), "UTF-8");
        try {
            polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script4, 15000);
            Assert.fail();
        } catch (Exception e) {
            Assert.assertEquals(TimeoutException.class, e.getClass());
        }
        String script5 = IOUtils.toString(ResourceUtils.getURL("classpath:script/runningFailScript.py").openStream(), "UTF-8");
        try {
            polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script5, 15000);
            Assert.fail();
        } catch (Exception e) {
            Assert.assertTrue(ExecutionException.class.equals(e.getClass()) || TimeoutException.class.equals(e.getClass()));
        }
        String script6 = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson2.py").openStream(), "UTF-8");
        Assert.assertEquals("错误的手机号码\n", polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script6, 30000));
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "print(1+1)", 5000), "2\n");
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/longScript.py").openStream(), "UTF-8");
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000), "Before sleep\n" + "After sleep\n");
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "hello```python\r\nprint(1+1)\r\n```world", 5000), "2\n");
        Assert.assertEquals(polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "```python\r\nprint(1+1)\r\n```", 5000), "2\n");
        String script2 = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson1.py").openStream(), "UTF-8");
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}\n", polyglotService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script2, 30000));
        executorService.shutdown();
        polyglotService.destroy();
    }

    @Test(expected = PolyglotException.class)
    public void testRelease() throws Exception {
        GenericObjectPool pool = EasyMock.createMock(GenericObjectPool.class);
        PolyglotService.PolyglotContext polyglotContext = new PolyglotService.PolyglotContext(false);
        EasyMock.expect(pool.borrowObject(1000)).andReturn(polyglotContext).anyTimes();
        pool.returnObject(polyglotContext);
        EasyMock.expectLastCall().anyTimes();
        pool.invalidateObject(polyglotContext);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(pool);
        try {
            PolyglotService.PyFuture pyFuture = new PolyglotService.PyFuture(pool, 1000, "JS");
            pyFuture.call();
        } finally {
            EasyMock.verify(pool);
        }
    }

    @Test
    public void testCheckEnv() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        PolyglotService pythonService = new PolyglotService();
        pythonService.setExecutorService(executorService);
        pythonService.setEmbedding(false);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkEnv.py").openStream(), "UTF-8");
        try {
            Assert.assertEquals("UNKNOWN\n" +
                    "{}\n" +
                    "UNKNOWN\n", pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        EasyMock.replay(executorService);
        PolyglotService.InitConfig service = new PolyglotService.InitConfig();
        service.setNotifierService(notifierManager);
        service.setEmbedding(false);
        service.setTimeout(100);
        service.setExecutorService(executorService);
        service.setTimeout4Corrector(200);
        service.setTimeout4Condition(100);
        PolyglotService empty = service.polyglotService();
        empty.init();
        Assert.assertEquals(service.getEmbedding(), empty.getEmbedding());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout4Condition());
        EasyMock.verify(executorService);
        Assert.assertNotNull(empty.getObjectPool());
        empty.setObjectPool(null);
        Assert.assertNull(empty.getObjectPool());
    }

    @Test
    public void testBuildEmpty() {
        PolyglotService pythonService = new PolyglotService();
        Assert.assertEquals("HELLO", pythonService.buildScript(null, "HELLO"));
    }

    @Test
    public void testBuildScriptWithEnv() throws Exception {
        PolyglotService service = new PolyglotService();
        ScriptEnv env = new ScriptEnv(ObjectBuilder.buildWorkflowTask());
        env.put("KEY", "VALUE");
        String script = service.buildScript(env, "print(1)");
        Assert.assertTrue(script.contains("os.environ[\"KEY\"] = \"VALUE\""));
    }

    @Test
    public void testCleanNoTags() throws Exception {
        PolyglotService service = new PolyglotService();
        String script = "  print(1)  ";
        Assert.assertEquals("print(1)", service.clean(script));
    }

    @Test(expected = Exception.class)
    public void testPyFutureBorrowFail() throws Exception {
        GenericObjectPool pool = EasyMock.createMock(GenericObjectPool.class);
        EasyMock.expect(pool.borrowObject(EasyMock.anyLong())).andThrow(new RuntimeException());
        EasyMock.replay(pool);
        PolyglotService.PyFuture future = new PolyglotService.PyFuture(pool, 100, "script");
        future.call();
    }

    @Test
    public void testPolyglotContextReset() throws Exception {
        PolyglotService.PolyglotContext context = new PolyglotService.PolyglotContext(false);
        context.reset();
        Assert.assertEquals("", context.content());
        context.close();
    }

    @Test
    public void testStringBufferOutputStreamWrite() throws Exception {
        PolyglotService.StringBufferOutputStream os = new PolyglotService.StringBufferOutputStream();
        os.write('A');
        os.write("BC".getBytes());
        os.write("DEFG".getBytes(), 1, 2);
        Assert.assertEquals("ABCEF", os.getBuffer().toString());
    }
}
