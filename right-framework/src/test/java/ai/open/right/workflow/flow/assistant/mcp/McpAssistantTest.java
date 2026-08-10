package ai.open.right.workflow.flow.assistant.mcp;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class McpAssistantTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = McpAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = McpAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testRuntime() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setDynamic("Dynamic");
        mcpConfig.setSuffix("Suffix");
        mcpConfig.setPrefix("Prefix");
        mcpConfig.setTimeout(100);
        McpAssistant mcpAssistant = new McpAssistant();
        McpRuntime mcpRuntime = mcpAssistant.buildMcpRuntime(mcpConfig, workflowTask);
        Assert.assertEquals(Integer.valueOf(100), mcpRuntime.getTimeout(1000));
        Assert.assertEquals(Integer.valueOf(100), mcpRuntime.getTimeout());
        Assert.assertEquals("Dynamic", mcpRuntime.getDynamic());
        Assert.assertEquals(workflowTask, mcpRuntime.getWorkTask());
        Assert.assertEquals("Suffix", mcpRuntime.getSuffix());
        Assert.assertEquals("Prefix", mcpRuntime.getPrefix());
    }

    @Test
    public void testBuildMcpDimension() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        McpAssistant mcpAssistant = new McpAssistant();
        McpDimension mcpDimension = McpDimension.builder().build();
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        EasyMock.expect(mcpDimensionService.buildDimension(mcpDimension, workflowTask)).andReturn(mcpDimension).anyTimes();
        EasyMock.replay(mcpDimensionService);
        mcpAssistant.setMcpDimensionService(mcpDimensionService);
        McpDimension mcpDimension2 = mcpAssistant.buildMcpDimension(mcpDimension, workflowTask);
        Assert.assertNotNull(mcpDimension2);
        EasyMock.verify(mcpDimensionService);
    }
}
