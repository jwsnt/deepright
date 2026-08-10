package ai.open.right.workflow.mcp.dimension.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.impl.McpDimensionServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class McpDimensionServiceImplTest {

    @Test
    public void testWithConfig() throws Exception {
        McpConfig mcpConfig = new McpConfig();
        mcpConfig.setClient("A");
        mcpConfig.setName("B");
        McpDimension dimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        McpDimensionServiceImpl mcpDimensionService = new McpDimensionServiceImpl();
        McpDimension target = mcpDimensionService.buildDimension(dimension, workTask);
        Assert.assertEquals("A", target.getClient());
        Assert.assertEquals("B", target.getName());
    }

    @Test
    public void testWithOutConfigWithDecode() throws Exception {
        NamesService namesService = EasyMock.createMock(NamesService.class);
        String code = NamesService.PREFIX_TOOLS + "UNKNOWN";
        EasyMock.expect(namesService.isPrefix(code)).andReturn(true).anyTimes();
        EasyMock.expect(namesService.decode(code)).andReturn(new String[]{"A1", "B1"}).anyTimes();
        EasyMock.replay(namesService);
        McpConfig mcpConfig = new McpConfig();
        McpDimension dimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        workTask.setWorkflow(code);
        McpDimensionServiceImpl mcpDimensionService = new McpDimensionServiceImpl();
        mcpDimensionService.setNamesService(namesService);
        McpDimension target = mcpDimensionService.buildDimension(dimension, workTask);
        Assert.assertEquals("A1", target.getClient());
        Assert.assertEquals("B1", target.getName());
        EasyMock.verify(namesService);
    }

    @Test
    public void testWithOutConfig() throws Exception {
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.expect(namesService.decode("UNKNOWN")).andReturn(new String[]{"A1", "B1"}).anyTimes();
        EasyMock.expect(namesService.isPrefix("UNKNOWN")).andReturn(false).anyTimes();
        EasyMock.replay(namesService);
        McpConfig mcpConfig = new McpConfig();
        McpDimension dimension = McpDimension.builder()
                .mcpConfig(mcpConfig)
                .build();
        WorkflowTask workTask = ObjectBuilder.buildWorkflowTask();
        McpDimensionServiceImpl mcpDimensionService = new McpDimensionServiceImpl();
        mcpDimensionService.setNamesService(namesService);
        McpDimension target = mcpDimensionService.buildDimension(dimension, workTask);
        Assert.assertNull(target.getClient());
        Assert.assertNull(target.getName());
        EasyMock.verify(namesService);
    }

    @Test
    public void testInit() throws Exception {
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        McpDimensionServiceImpl.InitConfig initConfig = new McpDimensionServiceImpl.InitConfig();
        initConfig.setNamesService(namesService);
        McpDimensionServiceImpl mcpDimensionService = initConfig.mcpDimensionService();
        Assert.assertEquals(namesService, mcpDimensionService.getNamesService());
        EasyMock.verify(namesService);
    }
}
