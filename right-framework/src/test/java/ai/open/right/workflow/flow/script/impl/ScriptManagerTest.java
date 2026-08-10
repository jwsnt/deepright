package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.script.ScriptConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class ScriptManagerTest {

    @Test
    public void testJs() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_JAVASCRIPT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(javaScriptService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(javaScriptService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setJavaScriptService(javaScriptService);
        String response = scriptManager.run(scriptConfig, workflowTask);
        Assert.assertEquals("HELLO", response);
        EasyMock.verify(javaScriptService);
    }

    @Test(expected = RuntimeException.class)
    public void testJsWithException() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_JAVASCRIPT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(javaScriptService.run(scriptConfig, workflowTask)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(javaScriptService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setJavaScriptService(javaScriptService);
        try {
            scriptManager.run(scriptConfig, workflowTask);
        } finally {
            EasyMock.verify(javaScriptService);
        }
    }

    @Test
    public void testPython() throws Exception {
        PythonService pyScriptService = EasyMock.createMock(PythonService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_PYTHON);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(pyScriptService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(pyScriptService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setPythonService(pyScriptService);
        String response = scriptManager.run(scriptConfig, workflowTask);
        Assert.assertEquals("HELLO", response);
        EasyMock.verify(pyScriptService);
    }

    @Test
    public void testJython() throws Exception {
        JythonService jyScriptService = EasyMock.createMock(JythonService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_JYTHON);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(jyScriptService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(jyScriptService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setJythonService(jyScriptService);
        String response = scriptManager.run(scriptConfig, workflowTask);
        Assert.assertEquals("HELLO", response);
        EasyMock.verify(jyScriptService);
    }

    @Test
    public void testLua() throws Exception {
        LuaService luaService = EasyMock.createMock(LuaService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_LUA);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(luaService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(luaService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setLuaService(luaService);
        String response = scriptManager.run(scriptConfig, workflowTask);
        Assert.assertEquals("HELLO", response);
        EasyMock.verify(luaService);
    }

    @Test
    public void testInit() throws Exception {
        JavaScriptService javaScriptService = EasyMock.createMock(JavaScriptService.class);
        PythonService pythonService = EasyMock.createMock(PythonService.class);
        JythonService jythonService = EasyMock.createMock(JythonService.class);
        CommandService commandService = EasyMock.createMock(CommandService.class);
        PolyglotService polyglotService = EasyMock.createMock(PolyglotService.class);
        LuaService luaService = EasyMock.createMock(LuaService.class);
        EasyMock.replay(polyglotService, commandService, javaScriptService, pythonService, jythonService, luaService);
        ScriptServiceImpl.InitConfig scriptManager = new ScriptServiceImpl.InitConfig();
        scriptManager.setJythonService(jythonService);
        scriptManager.setPythonService(pythonService);
        scriptManager.setJavaScriptService(javaScriptService);
        scriptManager.setLuaService(luaService);
        ScriptServiceImpl empty = (ScriptServiceImpl) scriptManager.scriptService();
        Assert.assertEquals(scriptManager.getCommandService(), empty.getCommandService());
        Assert.assertEquals(scriptManager.getPolyglotService(), empty.getPolyglotService());
        Assert.assertEquals(jythonService, empty.getJythonService());
        Assert.assertEquals(javaScriptService, empty.getJavaScriptService());
        Assert.assertEquals(pythonService, empty.getPythonService());
        Assert.assertEquals(luaService, empty.getLuaService());
        EasyMock.verify(polyglotService, commandService, javaScriptService, pythonService, jythonService, luaService);
    }

    @Test(expected = RuntimeException.class)
    public void testWithOutline() throws Exception {
        PythonService pythonService = EasyMock.createMock(PythonService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine("HELLO WORLD");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(pythonService.run(scriptConfig, workflowTask)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(pythonService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setPythonService(pythonService);
        try {
            // Without config PyService
            scriptManager.run(scriptConfig, workflowTask);
        } finally {
            EasyMock.verify(pythonService);
        }
    }

    @Test
    public void testWithCommand() throws Exception {
        CommandService commandService = EasyMock.createMock(CommandService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_COMMAND);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(commandService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(commandService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setCommandService(commandService);
        Assert.assertEquals("HELLO", scriptManager.run(scriptConfig, workflowTask));
        EasyMock.verify(commandService);
    }

    @Test
    public void testWithPolyglot() throws Exception {
        PolyglotService polyglotService = EasyMock.createMock(PolyglotService.class);
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setEngine(ScriptConfig.ENGINE_POLYGLOT);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(polyglotService.run(scriptConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(polyglotService);
        ScriptServiceImpl scriptManager = new ScriptServiceImpl();
        scriptManager.setPolyglotService(polyglotService);
        Assert.assertEquals("HELLO", scriptManager.run(scriptConfig, workflowTask));
        EasyMock.verify(polyglotService);
    }
}
