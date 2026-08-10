package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.store.history.impl.RedisHistoryStore;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.mozilla.javascript.EcmaError;
import org.mozilla.javascript.JavaScriptException;
import org.springframework.util.ResourceUtils;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeoutException;

public class JavaScriptServiceTest {

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
    public void testExample1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example1.js").openStream()), 10000);
        } catch (Exception e) {
            Assert.assertEquals("Mobile was error (JavaScriptCode#6)", e.getMessage());
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testExample2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example2.js").openStream()), 10000);
        Assert.assertEquals("{\"mobile\":12345678901,\"price\":100,\"currency\":\"NGN\"}", response);
        executorService.shutdown();
    }

    @Test
    public void testExample3() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example3.js").openStream()), 10000);
        Assert.assertEquals("{\"code\": 502, \"data\": {\"price\": 100, \"currency\": \"NGN\"}}", response);
        executorService.shutdown();
    }

    @Test
    public void testExample4() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example4.js").openStream()), 10000);
        Assert.assertEquals("{\"code\": 200, \"data\": {\"price\": 100, \"currency\": \"NGN\"}}", response);
        executorService.shutdown();
    }

    @Test(expected = TimeoutException.class)
    public void testTimeout() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example5.js").openStream()), 1);
        } catch (Exception e) {
            executorService.shutdown();
            throw e;
        }
    }

    @Test
    public void testExample6() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example6.js").openStream()), 10000);
        } catch (Exception e) {
            Assert.assertEquals("ai.open.right.WorkflowException: Undefined", e.getMessage());
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testExample7() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example7.js").openStream()), 10000);
        } catch (Exception e) {
            Assert.assertEquals("ReferenceError: “hello” 未定义。 (JavaScriptCode#1)", e.getMessage());
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testExampleWithExtract() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        Assert.assertEquals("{\"code\": 200, \"data\": {\"price\": 100, \"currency\": \"NGN\"}}", javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example8.js").openStream()), 10000));
        executorService.shutdown();
    }

    @Test
    public void testExample9() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example9.js").openStream()), 10000);
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}", response);
        executorService.shutdown();
    }

    @Test(expected = JavaScriptException.class)
    public void testExample10() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example10.js").openStream()), 10000);
        } catch (WorkflowException e) {
            Assert.assertEquals("错误的手机号码", e.getMessage());
            throw e;
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testMulti() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example1.js").openStream()), 20000);
        } catch (Exception e) {
            Assert.assertEquals("Mobile was error (JavaScriptCode#6)", e.getMessage());
        }
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example2.js").openStream()), 20000);
        Assert.assertEquals("{\"mobile\":12345678901,\"price\":100,\"currency\":\"NGN\"}", response);
        String response2 = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example3.js").openStream()), 20000);
        Assert.assertEquals("{\"code\": 502, \"data\": {\"price\": 100, \"currency\": \"NGN\"}}", response2);
        String response3 = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example4.js").openStream()), 20000);
        Assert.assertEquals("{\"code\": 200, \"data\": {\"price\": 100, \"currency\": \"NGN\"}}", response3);
        // Warn: Loop
        //  try {
        //      javaScriptService.run(IOUtils.toString(ResourceUtils.getURL("classpath:script/example5.js").openStream()), 1);
        //  } catch (Exception e) {
        //      Assert.assertEquals(TimeoutException.class, e.getClass());
        //  }
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example6.js").openStream()), 20000);
        } catch (Exception e) {
            Assert.assertEquals("ai.open.right.WorkflowException: Undefined", e.getMessage());
        }
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example7.js").openStream()), 20000);
        } catch (Exception e) {
            Assert.assertEquals("ReferenceError: “hello” 未定义。 (JavaScriptCode#1)", e.getMessage());
        }
        Assert.assertEquals("{\"code\": 200, \"data\": {\"price\": 100, \"currency\": \"NGN\"}}", javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example8.js").openStream()), 20000));
        String response4 = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example9.js").openStream()), 20000);
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}", response4);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example10.js").openStream()), 20000);
        } catch (Exception e) {
            Assert.assertEquals("错误的手机号码 (JavaScriptCode#21)", e.getMessage());
        }
        executorService.shutdown();
    }

    @Test(expected = EcmaError.class)
    public void testRelease() throws Exception {
        JavaScriptService.JsFuture jsFuture = new JavaScriptService().new JsFuture(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "JS");
        jsFuture.call();
    }

    @Test(expected = ExecutionException.class)
    public void testError() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        try {
            javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example11.js").openStream()), 10000);
        } catch (ExecutionException e) {
            Assert.assertEquals("{\"mobile\":12345678901112,\"value\":\"你好\"}", e.getCause().getMessage());
            throw e;
        } finally {
            executorService.shutdown();
        }
    }

    @Test
    public void testExample12() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example12.js").openStream()), 10000);
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}", response);
        executorService.shutdown();
    }

    @Test(expected = JavaScriptException.class)
    public void testExample13() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example13.js").openStream()), 10000);
        Assert.assertEquals("错误的手机号码", response);
        executorService.shutdown();
    }

    @Test
    public void testExample14() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        JavaScriptService javaScriptService = new JavaScriptService();
        javaScriptService.setExecutorService(executorService);
        String response = javaScriptService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), IOUtils.toString(ResourceUtils.getURL("classpath:script/example14.js").openStream()), 10000);
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}", response);
        executorService.shutdown();
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        EasyMock.replay(executorService);
        JavaScriptService.InitConfig service = new JavaScriptService.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout(100);
        service.setExecutorService(executorService);
        service.setTimeout4Corrector(200);
        service.setTimeout4Condition(100);
        JavaScriptService empty = service.javaScriptService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout4Condition());
        EasyMock.verify(executorService);
    }
    @Test(expected = WorkflowException.class)
    public void testGetObjectUndefined() {
        JavaScriptService service = new JavaScriptService();
        service.getObject(org.mozilla.javascript.Undefined.instance);
    }
}
