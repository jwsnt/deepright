package ai.open.right.workflow.flow.select.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.FunctionService;
import ai.open.right.workflow.flow.select.ChainSelectConfig;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class ChainSelectServiceImplTest {

    @Test
    public void testBuildChainFunction() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionService functionService = EasyMock.createMock(FunctionService.class);
        EasyMock.expect(functionService.call(chainSelectConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(functionService);
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setFunctionService(functionService);
        String chain = chainSelectService.buildChainFunction(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
        EasyMock.verify(functionService);
    }

    @Test
    public void testChainFunction() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionService functionService = EasyMock.createMock(FunctionService.class);
        EasyMock.expect(functionService.call(chainSelectConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(functionService);
        chainSelectConfig.setName("FUNCTION");
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setFunctionService(functionService);
        String chain = chainSelectService.selectChain(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
        EasyMock.verify(functionService);
    }

    @Test(expected = RuntimeException.class)
    public void testBuildChainFunctionWithException() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionService functionService = EasyMock.createMock(FunctionService.class);
        EasyMock.expect(functionService.call(chainSelectConfig, workflowTask)).andThrow(new RuntimeException("WORLD")).anyTimes();
        EasyMock.replay(functionService);
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setFunctionService(functionService);
        String chain = chainSelectService.buildChainFunction(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
        EasyMock.verify(functionService);
    }

    @Test(expected = RuntimeException.class)
    public void testChainFunctionWithException() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setName("FUNCTION");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionService functionService = EasyMock.createMock(FunctionService.class);
        EasyMock.expect(functionService.call(chainSelectConfig, workflowTask)).andThrow(new RuntimeException("WORLD")).anyTimes();
        EasyMock.replay(functionService);
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setFunctionService(functionService);
        String chain = chainSelectService.selectChain(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
        EasyMock.verify(functionService);
    }

    @Test
    public void testBuildChainDynamic() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setDynamic("DYNAMIC");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO"));
        String chain = chainSelectService.buildChainDynamic(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
    }

    @Test
    public void testChain() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setDynamic("DYNAMIC");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO"));
        String chain = chainSelectService.selectChain(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
    }

    @Test(expected = RuntimeException.class)
    public void testChainWithException() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setDynamic("DYNAMIC");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        chainSelectService.selectChain(chainSelectConfig, workflowTask);
    }

    @Test
    public void testChainWithExceptionAndDefault() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setDynamic("DYNAMIC");
        chainSelectConfig.setChain("HELLO");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        chainSelectService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        String chain = chainSelectService.selectChain(chainSelectConfig, workflowTask);
        Assert.assertEquals("HELLO", chain);
    }


    @Test
    public void testHashCode1() throws Exception {
        Object object = ChainSelectServiceImpl.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ChainSelectServiceImpl.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testInit() throws Exception {
        FunctionService functionService = EasyMock.createMock(FunctionService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        EasyMock.replay(functionService, notifierService);
        ChainSelectServiceImpl.InitConfig initConfig = new ChainSelectServiceImpl.InitConfig();
        initConfig.setFunctionService(functionService);
        initConfig.setNotifierService(notifierService);
        initConfig.setTimeout4Llm(10096);
        ChainSelectServiceImpl empty = (ChainSelectServiceImpl) initConfig.chainSelectService();
        Assert.assertEquals(initConfig.getFunctionService(), empty.getFunctionService());
        Assert.assertEquals(initConfig.getNotifierService(), empty.getNotifierService());
        Assert.assertEquals(initConfig.getTimeout4Llm(), empty.getTimeout4Llm());
        EasyMock.verify(functionService, notifierService);
    }

    @Test
    public void testDefault() throws Exception {
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setChain("HELLO");
        ChainSelectServiceImpl chainSelectService = new ChainSelectServiceImpl();
        Assert.assertEquals("HELLO", chainSelectService.selectChain(chainSelectConfig, ObjectBuilder.buildWorkflowTask()));
    }
}
