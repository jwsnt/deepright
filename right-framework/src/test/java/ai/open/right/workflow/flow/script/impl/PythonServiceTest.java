package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.nio.charset.StandardCharsets;

public class PythonServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = PythonService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = PythonService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testShortScript() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        Assert.assertEquals(pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "print(1+1)", 5000), "2\n");
    }

    @Test
    public void testLongScript() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/longScript.py").openStream(), "UTF-8");
        Assert.assertEquals(pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000), "Before sleep\n" + "After sleep\n");
    }

    @Test(expected = WorkflowException.class)
    public void testTimeoutScript() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(10000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/timeoutScript.py").openStream(), "UTF-8");
        pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 10000);
        Assert.fail();
    }

    @Test(expected = WorkflowException.class)
    public void testErrorScript() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/noModuleScript.py").openStream(), "UTF-8");
        pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000);
        Assert.fail();
    }

    @Test(expected = WorkflowException.class)
    public void testRunFailedScript() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/runningFailScript.py").openStream(), "UTF-8");
        pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 5000);
        Assert.fail();
    }

    @Test
    public void testLongTimeScript() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(30000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/sleepAndOutput.py").openStream(), "UTF-8");
        Assert.assertTrue(pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000).length() > 100);
    }

    @Test(expected = WorkflowException.class)
    public void testLongTimeErrorSleep() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(30000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/sleepAndError.py").openStream(), "UTF-8");
        pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000);
    }

    @Test
    public void testExtract() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        Assert.assertEquals(pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "hello```python\r\nprint(1+1)\r\n```world", 5000), "2\n");
    }

    @Test
    public void testClean() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        Assert.assertEquals(pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "```python\r\nprint(1+1)\r\n```", 5000), "2\n");
    }

    @Test
    public void testCheckJson1() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson1.py").openStream(), "UTF-8");
        Assert.assertEquals("{\"mobile\":1234567890,\"value\":\"你好\"}\n", pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
    }

    @Test
    public void testCheckJson2() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        String script = IOUtils.toString(ResourceUtils.getURL("classpath:script/checkJson2.py").openStream(), "UTF-8");
        Assert.assertEquals("错误的手机号码\n", pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 30000));
    }

    @Test
    public void testEnvScript1() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        ScriptEnv scriptEnv = new ScriptEnv(ObjectBuilder.buildWorkflowTask());
        scriptEnv.put("HELLO", "WORLD");
        Assert.assertEquals(pythonService.run(scriptEnv, "import os; print(os.environ.get(\"HELLO\"))", 5000), "WORLD\n");
    }

    @Test
    public void testEnvScript2() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setTimeout(5000);
        pythonService.setSegment(10);
        pythonService.init();
        ScriptEnv scriptEnv = new ScriptEnv(ObjectBuilder.buildWorkflowTask());
        scriptEnv.put("HELLO", "WORLD");
        Assert.assertEquals("UNKNOWN\n" +
                "{}\n" +
                "UNKNOWN\n", pythonService.run(scriptEnv, IOUtils.toString(ResourceUtils.getURL("classpath:script/checkEnv.py").openStream()), 5000));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        PythonService.InitConfig service = new PythonService.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTransfer(true);
        service.setPath("P");
        service.setPython("T");
        service.setHome("H");
        service.setTimeout(100);
        service.setSegment(1000);
        service.setTimeout4Corrector(200);
        service.setTimeout4Condition(100);
        PythonService empty = service.pythonService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(true, empty.getTransfer());
        Assert.assertEquals("T", empty.getPython());
        Assert.assertEquals("H", empty.getHome());
        Assert.assertEquals("P", empty.getPath());
        Assert.assertEquals(Integer.valueOf(1000), empty.getSegment());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout());
        Assert.assertEquals(Integer.valueOf(200), empty.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(100), empty.getTimeout4Condition());
    }
    @Test
    public void testBuildScriptContent() {
        PythonService service = new PythonService();
        String script = service.buildScript("print(1)");
        Assert.assertTrue(script.contains("import sys"));
    }

    /**
     * 覆盖 run 中 waitFor 超时分支：进程在 segment 内未结束时进入 else，
     * 若 {@code process.getInputStream().available() > 0} 则将已产生的 stdout 拷入 {@code inputWriter}。
     * 脚本先 write+flush 再 sleep，使首段 wait 返回 false 且管道中已有字节。
     */
    @Test
    public void run_copiesPartialStdoutWhenWaitForTimesOutPerSegment() throws Exception {
        PythonService pythonService = new PythonService();
        pythonService.setTransfer(System.getProperty("os.name").toLowerCase().contains("win"));
        pythonService.setPython("python3");
        pythonService.setSegment(10);
        pythonService.init();
        String script = "import sys,time\n"
                + "sys.stdout.write('PARTIAL_SEGMENT_OUT\\n')\n"
                + "sys.stdout.flush()\n"
                + "time.sleep(3)\n"
                + "print('FINAL_SEGMENT_OUT')\n";
        String out = pythonService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), script, 15000);
        Assert.assertTrue("stdout should contain early partial line: " + out, out.contains("PARTIAL_SEGMENT_OUT"));
        Assert.assertTrue("stdout should contain final line: " + out, out.contains("FINAL_SEGMENT_OUT"));
    }
}
