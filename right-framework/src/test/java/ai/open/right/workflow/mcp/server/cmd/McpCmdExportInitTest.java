package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.ConfigSearch;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.impl.WorkflowConfigServiceImpl;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

public class McpCmdExportInitTest {

    @Test
    public void test() throws Exception {
        Map<String, Object> config = JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class);
        McpExportConfig mcpExportConfig = new McpExportConfig();
        McpCmdExportService mcpCmdExportService = EasyMock.createMock(McpCmdExportService.class);
        mcpCmdExportService.export(mcpExportConfig);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(mcpCmdExportService);
        McpCmdExportConfigInit mcpCmdExportInit = new McpCmdExportConfigInit();
        mcpCmdExportInit.setWorkflowConfigService(new WorkflowConfigServiceImpl() {
            public WorkflowConfig config(ConfigSearch configSearch, String workflow) throws Exception {
                McpConfig mcpConfig = new McpConfig();
                mcpConfig.setExportConfig(mcpExportConfig);
                WorkflowConfig workflowConfig = new WorkflowConfig();
                workflowConfig.setMcpConfig(mcpConfig);
                return workflowConfig;
            }
        });
        mcpCmdExportInit.setMcpCmdExportServices(Arrays.asList(mcpCmdExportService));
        mcpCmdExportInit.init(config);
        EasyMock.verify(mcpCmdExportService);
    }

    @Test
    public void testInit() throws Exception {
        List<McpCmdExportService> exportServices = new ArrayList<McpCmdExportService>();
        WorkflowConfigServiceImpl workflowConfigService = new WorkflowConfigServiceImpl();
        McpCmdExportConfigInit.InitConfig initConfig = new McpCmdExportConfigInit.InitConfig();
        initConfig.setMcpCmdExportServices(exportServices);
        initConfig.setWorkflowConfigService(workflowConfigService);
        McpCmdExportConfigInit mcpCmdExportInit = initConfig.mcpCmdExportConfigInit();
        Assert.assertEquals(exportServices, mcpCmdExportInit.getMcpCmdExportServices());
        Assert.assertEquals(workflowConfigService, mcpCmdExportInit.getWorkflowConfigService());
    }
}
