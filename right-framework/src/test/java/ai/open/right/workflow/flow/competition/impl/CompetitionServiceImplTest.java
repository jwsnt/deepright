package ai.open.right.workflow.flow.competition.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.condition.Condition;
import ai.open.right.workflow.flow.competition.CompetitionConfig;
import ai.open.right.workflow.flow.competition.ConditionConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class CompetitionServiceImplTest {

    @Test
    public void test() throws Exception {
        CompetitionConfig competitionConfig = new CompetitionConfig();
        ConditionConfig conditionConfig = new ConditionConfig();
        conditionConfig.setCondition("Condition1");
        conditionConfig.setDynamic("Target1");
        competitionConfig.setConditionConfigs(Arrays.asList(conditionConfig));
        CompetitionServiceImpl competitionService = new CompetitionServiceImpl();
        competitionService.setTimeout(1000);
        competitionService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("TRUE"));
        String workflow = competitionService.compete(competitionConfig, ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals("Target1", workflow);
    }

    @Test
    public void testWithExceptionAndStopOnFailedFalse() throws Exception {
        CompetitionConfig competitionConfig = new CompetitionConfig();
        ConditionConfig conditionConfig = new ConditionConfig();
        conditionConfig.setCondition("Condition1");
        conditionConfig.setDynamic("Target1");
        competitionConfig.setDynamic("Target2");
        competitionConfig.setConditionConfigs(Arrays.asList(conditionConfig));
        CompetitionServiceImpl competitionService = new CompetitionServiceImpl() {
            @Override
            protected Condition checkCondition(ConditionTask task) {
                throw new RuntimeException();
            }
        };
        competitionService.setTimeout(1000);
        competitionService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("TRUE"));
        String workflow = competitionService.compete(competitionConfig, ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals("Target2", workflow);
    }

    @Test(expected = RuntimeException.class)
    public void testWithExceptionAndStopOnFailedTrue() throws Exception {
        CompetitionConfig competitionConfig = new CompetitionConfig();
        ConditionConfig conditionConfig = new ConditionConfig();
        conditionConfig.setCondition("Condition1");
        competitionConfig.setStopOnFailed(true);
        conditionConfig.setDynamic("Target1");
        competitionConfig.setConditionConfigs(Arrays.asList(conditionConfig));
        CompetitionServiceImpl competitionService = new CompetitionServiceImpl() {
            @Override
            protected Condition checkCondition(ConditionTask task) {
                throw new RuntimeException();
            }
        };
        competitionService.setTimeout(1000);
        competitionService.setNotifierService(ObjectBuilder.buildAssertNotifierManagerWithWriteBackDirect("UNKNOWN"));
        competitionService.compete(competitionConfig, ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        CompetitionServiceImpl.InitConfig competitionService = new CompetitionServiceImpl.InitConfig();
        competitionService.setNotifierService(notifierManager);
        competitionService.setTimeout(1000);
        CompetitionServiceImpl empty = (CompetitionServiceImpl) competitionService.competitionService();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
    }
}
