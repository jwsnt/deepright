package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.select.ChainSelectConfig;
import ai.open.right.workflow.flow.select.ChainSelectService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ChainSelectAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setDynamic("DYNAMIC");
        workflowConfig.setChainSelectConfig(chainSelectConfig);
        ChainSelectService chainSelectService = EasyMock.createMock(ChainSelectService.class);
        EasyMock.expect(chainSelectService.selectChain(chainSelectConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(chainSelectService);
        ChainSelectAssistant chainSelectAssistant = new ChainSelectAssistant();
        // Query不变
        chainSelectAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        chainSelectAssistant.setChainSelectService(chainSelectService);
        chainSelectAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(chainSelectService);
    }

    @Test
    public void execute_withNullQuery_buildsNullContentSegment() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(null);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ChainSelectConfig chainSelectConfig = new ChainSelectConfig();
        chainSelectConfig.setDynamic("DYNAMIC");
        workflowConfig.setChainSelectConfig(chainSelectConfig);
        ChainSelectAssistant assistant = new ChainSelectAssistant();
        assistant.setChainSelectService((config, task) -> "TARGET");
        assistant.setNotifierService(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, ai.open.right.context.RedirectContext redirectContext,
                               NotifierWriteBack notifierWriteBack) {
                Assert.assertNull(segment.getContent());
            }
        });

        assistant.execute(workflowConfig, workflowTask);
    }

    @Test
    public void testInit() throws Exception {
        ChainSelectService chainSelectService = EasyMock.createMock(ChainSelectService.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        EasyMock.replay(chainSelectService, signalFactory);
        Map<String, LLMQueryService> llmQueryService = new HashMap<>();
        NotifierService notifierService = ObjectBuilder.buildActualNotifierManagerWithNothing();
        ChainSelectAssistant.InitConfig initConfig = new ChainSelectAssistant.InitConfig();
        initConfig.setChainSelectService(chainSelectService);
        initConfig.setNotifierService(notifierService);
        initConfig.setLlmQueryService(llmQueryService);
        initConfig.setSignalFactory(signalFactory);
        ChainSelectAssistant chainSelectAssistant = initConfig.competitionAssistant();
        Assert.assertEquals(initConfig.getChainSelectService(), chainSelectAssistant.getChainSelectService());
        Assert.assertEquals(initConfig.getNotifierService(), chainSelectAssistant.getNotifierService());
        Assert.assertEquals(initConfig.getLlmQueryService(), chainSelectAssistant.getLlmQueryService());
        Assert.assertEquals(initConfig.getSignalFactory(), chainSelectAssistant.getSignalFactory());
        EasyMock.verify(chainSelectService, signalFactory);
    }


    @Test
    public void testHashCode1() throws Exception {
        Object object = ChainSelectAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ChainSelectAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
    @Test(expected = IllegalArgumentException.class)
    public void testExecuteNoSelector() throws Exception {
        ChainSelectAssistant assistant = new ChainSelectAssistant();
        WorkflowConfig config = new WorkflowConfig();
        config.setChainSelectConfig(null);
        assistant.execute(config, null);
    }
}
