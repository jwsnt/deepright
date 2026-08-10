package ai.open.right.workflow.flow.competition.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.competition.ConditionConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class ConditionTaskTest {

    @Test
    void testBuilderAndGetters() {
        SyncWorkflowTask syncTask = new SyncWorkflowTask(ObjectBuilder.buildWorkflowTask(), null, 1000);
        ConditionConfig conditionConfig = new ConditionConfig();
        CompetitionServiceImpl.ConditionTask conditionTask = CompetitionServiceImpl.ConditionTask.builder()
                .syncWorkflowTask(syncTask)
                .conditionConfig(conditionConfig)
                .build();
        assertSame(syncTask, conditionTask.getSyncWorkflowTask());
        assertSame(conditionConfig, conditionTask.getConditionConfig());
    }

    @Test
    void testSetters() {
        CompetitionServiceImpl.ConditionTask conditionTask = CompetitionServiceImpl.ConditionTask.builder().build();
        SyncWorkflowTask newSyncTask = new SyncWorkflowTask(ObjectBuilder.buildWorkflowTask(), null, 1000);
        ConditionConfig newConditionConfig = new ConditionConfig();
        conditionTask.setSyncWorkflowTask(newSyncTask);
        conditionTask.setConditionConfig(newConditionConfig);
        assertSame(newSyncTask, conditionTask.getSyncWorkflowTask());
        assertSame(newConditionConfig, conditionTask.getConditionConfig());
    }

    @Test
    void testNullValues() {
        CompetitionServiceImpl.ConditionTask conditionTask = CompetitionServiceImpl.ConditionTask.builder().build();
        assertNull(conditionTask.getSyncWorkflowTask());
        assertNull(conditionTask.getConditionConfig());
        conditionTask.setSyncWorkflowTask(null);
        conditionTask.setConditionConfig(null);
        assertNull(conditionTask.getSyncWorkflowTask());
        assertNull(conditionTask.getConditionConfig());
    }
}