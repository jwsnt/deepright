package ai.open.right.workflow.mcp.server.impl;

import ai.open.right.workflow.mcp.server.McpCmdExportService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class McpRequestServiceImplInitConfigTest {

    @Test
    public void shouldCreateMcpTaskServiceWithProvidedProperties() throws Exception {
        McpDistributorImpl.InitConfig init = new McpDistributorImpl.InitConfig();

        Map<String, McpCmdExportService> protocolServices = new HashMap<>();
        McpCmdExportService mockService = EasyMock.createMock(McpCmdExportService.class);
        protocolServices.put("test", mockService);
        init.setCmdServices(protocolServices);

        EasyMock.replay(mockService);

        // 使用反射设置属性
        try {
            java.lang.reflect.Field field = init.getClass().getDeclaredField("protocolServices");
            field.setAccessible(true);
            field.set(init, protocolServices);
        } catch (Exception e) {
            // 如果反射失败，直接测试bean创建
        }
        McpDistributorImpl bean = (McpDistributorImpl) init.mcpRequestService();
        Assert.assertEquals(init.getCmdServices(), bean.getCmdServices());
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof McpDistributorImpl);

        EasyMock.verify(mockService);
    }

    @Test
    public void shouldCreateMcpTaskServiceWithDefaults() throws Exception {
        McpDistributorImpl.InitConfig init = new McpDistributorImpl.InitConfig();
        Map<String, McpCmdExportService> protocolServices = new HashMap<>();
        init.setCmdServices(protocolServices);
        // 使用反射设置属性
        try {
            java.lang.reflect.Field field = init.getClass().getDeclaredField("protocolServices");
            field.setAccessible(true);
            field.set(init, protocolServices);
        } catch (Exception e) {
            // 如果反射失败，直接测试bean创建
        }
        McpDistributorImpl bean = (McpDistributorImpl) init.mcpRequestService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof McpDistributorImpl);
        Assert.assertEquals(protocolServices, bean.getCmdServices());
    }
}
