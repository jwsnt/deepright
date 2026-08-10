package ai.open.right.workflow.flow.function;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.function.impl.FunctionServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;

public class FunctionServiceImplTest {

    @Test
    public void test() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setName("F1");
        Function function = new Function() {

            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                Assert.assertEquals(workflowTask, functionContext.getWorkTask());
                Assert.assertEquals(functionConfig, functionContext.getFunctionConfig());
                return "OK";
            }
        };
        FunctionServiceImpl functionManager = new FunctionServiceImpl();
        functionManager.setFunctions(Collections.singletonMap("F1", function));
        Assert.assertEquals("OK", functionManager.call(functionConfig, workflowTask));
    }

    @Test
    public void testWithNull() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        FunctionConfig functionConfig = new FunctionConfig();
        functionConfig.setName("F1");
        Function function = new Function() {

            @Override
            public Object call(FunctionContext functionContext) throws WorkflowException {
                return null;
            }
        };
        FunctionServiceImpl functionManager = new FunctionServiceImpl();
        functionManager.setFunctions(Collections.singletonMap("F1", function));
        Assert.assertEquals("", functionManager.call(functionConfig, workflowTask));
    }
}
