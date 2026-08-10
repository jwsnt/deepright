package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.io.IOException;

public class CommandServiceTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = CommandService.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = CommandService.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testShortScript() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.init();
        Assert.assertEquals(commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "echo 1", 5000), "1\n");
    }

    @Test
    public void testLongScript() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.init();
        Assert.assertEquals(commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "sleep 3", 5000), "");
    }

    @Test(expected = WorkflowException.class)
    public void testTimeoutScript() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.init();
        commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "sleep 100", 10000);
        Assert.fail();
    }

    @Test(expected = IOException.class)
    public void testErrorScript() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.init();
        commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "exit 1", 5000);
        Assert.fail();
    }

    @Test(expected = WorkflowException.class)
    public void testErrorOutput() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.init();
        commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "sleep 10 && echo 1", 5000);
    }

    @Test
    public void testLargeOutput() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(10000);
        commandService.setSegment(10);
        commandService.init();
        commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "curl https://github.com/", 20000);
    }

    @Test
    public void testClean() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.init();
        Assert.assertEquals(commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "```\r\nsleep 2\r\n```", 5000), "");
    }

    @Test
    public void testPath() throws Exception {
        CommandService commandService = new CommandService();
        commandService.setTimeout(5000);
        commandService.setSegment(10);
        commandService.setHome("/bin");
        commandService.init();
        Assert.assertEquals(commandService.run(new ScriptEnv(ObjectBuilder.buildWorkflowTask()), "echo 1", 5000), "1\n");
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierService = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent();
        CommandService.InitConfig initConfig = new CommandService.InitConfig();
        initConfig.setHome("HOME");
        initConfig.setSegment(100);
        initConfig.setNotifierService(notifierService);
        initConfig.setTimeout(10);
        initConfig.setTimeout4Corrector(100);
        initConfig.setTimeout4Condition(1000);
        CommandService commandService = initConfig.commandService();
        Assert.assertEquals("HOME", commandService.getHome());
        Assert.assertEquals(Integer.valueOf(100), commandService.getSegment());
        Assert.assertEquals(notifierService, commandService.getNotifierService());
        Assert.assertEquals(Integer.valueOf(10), commandService.getTimeout());
        Assert.assertEquals(Integer.valueOf(100), commandService.getTimeout4Corrector());
        Assert.assertEquals(Integer.valueOf(1000), commandService.getTimeout4Condition());
    }
}
