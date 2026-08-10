package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.competition.CompetitionConfig;
import ai.open.right.workflow.flow.competition.CompetitionService;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class CompetitionAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowConfig workflowConfig = new WorkflowConfig();
        CompetitionConfig competitionConfig = new CompetitionConfig();
        workflowConfig.setCompetitionConfig(competitionConfig);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        CompetitionService competitionService = EasyMock.createMock(CompetitionService.class);
        EasyMock.expect(competitionService.compete(competitionConfig, workflowTask)).andReturn("HELLO").anyTimes();
        EasyMock.replay(competitionService);
        CompetitionAssistant competitionAssistant = new CompetitionAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
            }
        };
        competitionAssistant.setCompetitionService(competitionService);
        competitionAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(competitionService);
    }


    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        CompetitionService competitionService = EasyMock.createMock(CompetitionService.class);
        EasyMock.replay(competitionService, notifierManager, signalFactory);
        CompetitionAssistant.InitConfig assistant = new CompetitionAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setCompetitionService(competitionService);
        CompetitionAssistant empty = assistant.competitionAssistant();
        Assert.assertEquals(competitionService, empty.getCompetitionService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(competitionService, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = CompetitionAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = CompetitionAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
