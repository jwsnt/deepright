package ai.open.right.workflow.flow.function;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import org.junit.Assert;
import org.junit.Test;

public class FunctionContextTest {

    @Test
    public void testGetSet() {
        FunctionConfig functionConfig = new FunctionConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionContext functionContext = FunctionContext.builder()
                .functionConfig(functionConfig)
                .workTask(workflowTask)
                .build();
        Assert.assertEquals(functionConfig, functionContext.getFunctionConfig());
        Assert.assertEquals(workflowTask, functionContext.getWorkTask());
    }
}
