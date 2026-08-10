package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class TokenAssistantTest {

    @Test
    public void test() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        TokenStatistic tokenStatistic = EasyMock.createMock(TokenStatistic.class);
        EasyMock.expect(tokenStatistic.read(workflowTask)).andReturn(TokenData.builder().cache(1).build());
        EasyMock.replay(tokenStatistic);
        TokenAssistant tokenAssistant = new TokenAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                Assert.assertEquals("{\"thinking\":0,\"input\":0,\"total\":0,\"cache\":1}", content);
            }
        };
        tokenAssistant.setTokenStatistic(tokenStatistic);
        tokenAssistant.execute(new WorkflowConfig(), workflowTask);
        EasyMock.verify(tokenStatistic);
    }

    @Test
    public void testInit() throws Exception {
        TokenStatistic tokenStatistic = EasyMock.createMock(TokenStatistic.class);
        EasyMock.replay(tokenStatistic);
        TokenAssistant.InitConfig initConfig = new TokenAssistant.InitConfig();
        initConfig.setTokenStatistic(tokenStatistic);
        TokenAssistant empty = initConfig.tokenAssistant();
        Assert.assertEquals(tokenStatistic, empty.getTokenStatistic());
        EasyMock.verify(tokenStatistic);
    }
}
