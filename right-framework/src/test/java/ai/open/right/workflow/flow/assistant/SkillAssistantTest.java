package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.skill.SkillsFetcher;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class SkillAssistantTest {

    @Test
    public void test1() throws Exception {
        SkillsFetcher skillFetcher = EasyMock.createMock(SkillsFetcher.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.transfer(ImmutableMap.of("skill", "my_path"), String.class));
        EasyMock.expect(skillFetcher.fetchResource(workflowTask, "my_path", null)).andReturn("HELLO").anyTimes();
        EasyMock.replay(skillFetcher);
        SkillsAssistant skillAssistant = new SkillsAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("HELLO", content);
            }
        };
        skillAssistant.setSkillFetcher(skillFetcher);
        skillAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(skillFetcher);
    }

    @Test
    public void test2() throws Exception {
        SkillsFetcher skillFetcher = EasyMock.createMock(SkillsFetcher.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.transfer(ImmutableMap.of("skill", "my_path", "resource", "data"), String.class));
        EasyMock.expect(skillFetcher.fetchResource(workflowTask, "my_path", "data")).andReturn("HELLO").anyTimes();
        EasyMock.replay(skillFetcher);
        SkillsAssistant skillAssistant = new SkillsAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("HELLO", content);
            }
        };
        skillAssistant.setSkillFetcher(skillFetcher);
        skillAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(skillFetcher);
    }

    @Test
    public void testInit() throws Exception {
        SkillsFetcher skillFetcher = EasyMock.createMock(SkillsFetcher.class);
        SkillsAssistant.InitConfig initConfig = new SkillsAssistant.InitConfig();
        initConfig.setSkillFetcher(skillFetcher);
        SkillsAssistant assistant = initConfig.skillsAssistant();
        Assert.assertEquals(skillFetcher, assistant.getSkillFetcher());
    }

    @Test
    public void testExecuteNullMetadata() throws Exception {
        SkillsAssistant assistant = new SkillsAssistant();
        assistant.setNotifierService(ObjectBuilder.buildNotifierManagerWithimplement());
        SkillsFetcher fetcher = EasyMock.createMock(SkillsFetcher.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        EasyMock.expect(fetcher.fetchResource(workflowTask, "my_path", null)).andReturn("").anyTimes();
        EasyMock.expect(fetcher.fetchSkills(ObjectBuilder.buildWorkflowTask())).andReturn(null).anyTimes();
        EasyMock.replay(fetcher);
        assistant.setSkillFetcher(fetcher);
        workflowTask.setQuery(JsonUtils.transfer(ImmutableMap.of("skill", "my_path"), String.class));
        workflowTask.setNotifier(Notifier.ENDPOINT);
        assistant.execute(new WorkflowConfig(), workflowTask);
    }

    @Test
    public void testWhy() throws Exception {
        SkillsAssistant.SkillOperation skillOperation = new SkillsAssistant.SkillOperation();
        skillOperation.setWhy("WHY");
        Assert.assertEquals(skillOperation.getWhy(), "WHY");
    }
}
