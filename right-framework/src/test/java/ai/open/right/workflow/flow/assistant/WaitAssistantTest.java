package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class WaitAssistantTest {

    /** 覆盖 buildWait：wait != null 且 wait < max，返回 wait。 */
    @Test
    public void buildWait_waitLessThanMax_returnsWait() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(1000);
        
        Integer result = assistant.buildWait(workTask);
        
        Assert.assertEquals(Integer.valueOf(1000), result);
    }

    /** 覆盖 buildWait：wait != null 且 wait > max，返回 max。 */
    @Test
    public void buildWait_waitGreaterThanMax_returnsMax() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(10000);
        
        Integer result = assistant.buildWait(workTask);
        
        Assert.assertEquals(Integer.valueOf(5000), result);
    }

    /** 覆盖 buildWait：wait != null 且 wait == max，返回 max。 */
    @Test
    public void buildWait_waitEqualsMax_returnsMax() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(5000);
        
        Integer result = assistant.buildWait(workTask);
        
        Assert.assertEquals(Integer.valueOf(5000), result);
    }

    /** 覆盖 buildWait：wait == null，返回 max。 */
    @Test
    public void buildWait_waitIsNull_returnsMax() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(null);
        
        Integer result = assistant.buildWait(workTask);
        
        Assert.assertEquals(Integer.valueOf(5000), result);
    }

    /** 覆盖 doWait：执行等待并返回 wait 值。 */
    @Test
    public void doWait_sleepsAndReturnsWait() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        Integer wait = 10;
        
        long startTime = System.currentTimeMillis();
        Integer result = assistant.doWait(workTask, wait);
        long endTime = System.currentTimeMillis();
        
        Assert.assertEquals(wait, result);
        Assert.assertTrue("Should sleep at least " + wait + " ms", (endTime - startTime) >= wait);
    }

    /** 覆盖 buildContent：构建正确的内容字符串。 */
    @Test
    public void buildContent_buildsCorrectMessage() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        Integer wait = 1000;
        
        String result = assistant.buildContent(workTask, wait);
        
        Assert.assertEquals("Waited " + wait + " ms successfully", result);
    }

    /** 覆盖 execute：完整执行流程，wait < max。 */
    @Test
    public void execute_waitLessThanMax_callsChainOr2EndpointWithCorrectContent() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(1000);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        
        final String[] capturedContent = new String[1];
        WaitAssistant testAssistant = new WaitAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                capturedContent[0] = content;
            }
        };
        testAssistant.setMax(5000);
        
        testAssistant.execute(workflowConfig, workTask);
        
        Assert.assertNotNull(capturedContent[0]);
        Assert.assertEquals("Waited 1000 ms successfully", capturedContent[0]);
    }

    /** 覆盖 execute：完整执行流程，wait > max。 */
    @Test
    public void execute_waitGreaterThanMax_callsChainOr2EndpointWithMaxWait() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(10000);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        
        final String[] capturedContent = new String[1];
        WaitAssistant testAssistant = new WaitAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                capturedContent[0] = content;
            }
        };
        testAssistant.setMax(5000);
        
        testAssistant.execute(workflowConfig, workTask);
        
        Assert.assertNotNull(capturedContent[0]);
        Assert.assertEquals("Waited 5000 ms successfully", capturedContent[0]);
    }

    /** 覆盖 execute：完整执行流程，wait == null。 */
    @Test
    public void execute_waitIsNull_callsChainOr2EndpointWithMaxWait() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        assistant.setMax(5000);
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setObjectQuery(null);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        
        final String[] capturedContent = new String[1];
        WaitAssistant testAssistant = new WaitAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
                capturedContent[0] = content;
            }
        };
        testAssistant.setMax(5000);
        
        testAssistant.execute(workflowConfig, workTask);
        
        Assert.assertNotNull(capturedContent[0]);
        Assert.assertEquals("Waited 5000 ms successfully", capturedContent[0]);
    }

    /** 覆盖 InitConfig.waitAssistant：创建并配置 WaitAssistant。 */
    @Test
    public void testInitConfig() throws Exception {
        WaitAssistant.InitConfig initConfig = new WaitAssistant.InitConfig();
        initConfig.setMax(3000);
        WaitAssistant assistant = initConfig.waitAssistant();
        
        Assert.assertNotNull(assistant);
        Assert.assertEquals(Integer.valueOf(3000), assistant.getMax());
    }

    /** 覆盖 buildContent：wait 为 0 的情况。 */
    @Test
    public void buildContent_waitIsZero_buildsCorrectMessage() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        Integer wait = 0;
        
        String result = assistant.buildContent(workTask, wait);
        
        Assert.assertEquals("Waited 0 ms successfully", result);
    }

    /** 覆盖 doWait：wait 为 0 的情况，应该立即返回。 */
    @Test
    public void doWait_waitIsZero_returnsImmediately() throws Exception {
        WaitAssistant assistant = new WaitAssistant();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        Integer wait = 0;
        
        long startTime = System.currentTimeMillis();
        Integer result = assistant.doWait(workTask, wait);
        long endTime = System.currentTimeMillis();
        
        Assert.assertEquals(wait, result);
        Assert.assertTrue("Should return quickly when wait is 0", (endTime - startTime) < 100);
    }
}
