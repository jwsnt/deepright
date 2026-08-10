package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.python.core.PyException;
import org.python.util.PythonInterpreter;
import org.springframework.util.ResourceUtils;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeoutException;

public class JythonServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = JythonService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = JythonService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testShortScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.setTimeout(5000);
        jythonService.init();
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "print(1+1)", 5000), "2\n");
        executorService.shutdown();
    }

    @Test
    public void testLongScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/longScript.py").openStream(), "UTF-8");
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000), "Before sleep\n" +
                "After sleep\n");
        executorService.shutdown();
    }

    @Test(expected = TimeoutException.class)
    public void testTimeoutScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/timeoutScript.py").openStream(), "UTF-8");
        try {
            jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 10000);
            Assert.fail();
        } finally {
            executorService.shutdown();
        }
    }

    @Test(expected = ExecutionException.class)
    public void testErrorScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/noModuleScript.py").openStream(), "UTF-8");
        try {
            jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 10000);
            Assert.fail();
        } finally {
            executorService.shutdown();
        }
    }

    @Test(expected = ExecutionException.class)
    public void testRunFailedScript() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/runningFailScript.py").openStream(), "UTF-8");
        try {
            jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000);
            Assert.fail();
        } finally {
            executorService.shutdown();
        }
    }

    @Test(expected = ExecutionException.class)
    public void testLongTimeErrorSleep() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/sleepAndError.py").openStream(), "UTF-8");
        jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000);
        executorService.shutdown();
    }

    @Test
    public void testExtract() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "hello```python\r\nprint(1+1)\r\n```world", 5000), "2\n");
        executorService.shutdown();
    }

    @Test
    public void testClean() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "```python\r\nprint(1+1)\r\n```", 5000), "2\n");
        executorService.shutdown();
    }

    @Test
    public void testCheckJson1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson1_jy.py").openStream(), "UTF-8");
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}\n", jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
        executorService.shutdown();
    }

    @Test
    public void testCheckJson2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson2_jy.py").openStream(), "UTF-8");
        try {
            Assert.assertEquals("错误的手机号码\n", jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testMulti() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script3 = IOUtils.toString(ResourceUtils.getURL("classpath:script/timeoutScript.py").openStream(), "UTF-8");
        try {
            jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script3, 10000);
            Assert.fail();
        } catch (Exception e) {
            Assert.assertEquals(TimeoutException.class, e.getClass());
        }
        String script4 = IOUtils.toString(ResourceUtils.getURL("classpath:script/noModuleScript.py").openStream(), "UTF-8");
        try {
            jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script4, 15000);
            Assert.fail();
        } catch (Exception e) {
            Assert.assertEquals(ExecutionException.class, e.getClass());
        }
        String script5 = IOUtils.toString(ResourceUtils.getURL("classpath:script/runningFailScript.py").openStream(), "UTF-8");
        try {
            jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script5, 15000);
            Assert.fail();
        } catch (Exception e) {
            Assert.assertEquals(ExecutionException.class, e.getClass());
        }
        String script6 = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson2_jy.py").openStream(), "UTF-8");
        Assert.assertEquals("错误的手机号码\n", jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script6, 30000));
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "print(1+1)", 5000), "2\n");
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/longScript.py").openStream(), "UTF-8");
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000), "Before sleep\n" + "After sleep\n");
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "hello```python\r\nprint(1+1)\r\n```world", 5000), "2\n");
        Assert.assertEquals(jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "```python\r\nprint(1+1)\r\n```", 5000), "2\n");
        String script2 = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson1_jy.py").openStream(), "UTF-8");
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}\n", jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script2, 30000));
        executorService.shutdown();
    }

    @Test(expected = PyException.class)
    public void testRelease() throws Exception {
        GenericObjectPool pool = EasyMock.createMock(GenericObjectPool.class);
        PythonInterpreter pythonInterpreter = new PythonInterpreter();
        EasyMock.expect(pool.borrowObject(1000)).andReturn(pythonInterpreter).anyTimes();
        pool.returnObject(pythonInterpreter);
        EasyMock.expectLastCall().anyTimes();
        pool.invalidateObject(pythonInterpreter);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(pool);
        try {
            JythonService.JyFuture jyFuture = new JythonService.JyFuture(pool, new ScriptEnv(ObjectBuilder.buildWorkflowTask()), 1000, "JS");
            jyFuture.call();
        } finally {
            EasyMock.verify(pool);
        }
    }

    @Test
    public void testCheckEnv() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JythonService jythonService = new JythonService();
        jythonService.setExecutorService(executorService);
        jythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkEnv_jy.py").openStream(), "UTF-8");
        try {
            Assert.assertEquals("UNKNOWN\n" +
                    "{}\n" +
                    "UNKNOWN\n", jythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        EasyMock.replay(executorService);
        JythonService.InitConfig service = new JythonService.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout(100);
        service.setExecutorService(executorService);
        service.setTimeout4Corrector(200);
        service.setTimeout4Condition(100);
        JythonService empty = service.jythonService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout4Condition());
        EasyMock.verify(executorService);
    }
    @Test
    public void testAppendCompatibility() {
        JythonService service = new JythonService();
        String script = service.append("print(1)");
        Assert.assertTrue(script.contains(JythonService.COMPATIBILITY_1));
        Assert.assertTrue(script.contains(JythonService.COMPATIBILITY_2));
    }

    @Test
    public void testCleanNoTags() {
        JythonService service = new JythonService();
        Assert.assertEquals("print(1)", service.clean("  print(1)  "));
    }
}
