package ai.open.right.workflow.mcp.config.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.mcp.config.McpConfigInit;
import org.apache.commons.io.IOUtils;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Map;

public class McpConfigServiceImplTest {

    @Test
    public void testConfig() throws Exception {
        String config = IOUtils.toString(new BufferedInputStream(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream()), StandardCharsets.UTF_8);
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.expect(placeholderResolver.replace(config)).andReturn(config).anyTimes();
        EasyMock.replay(placeholderResolver);
        McpConfigServiceImpl mcpConfigService = new McpConfigServiceImpl();
        mcpConfigService.setPlaceholderResolver(placeholderResolver);
        mcpConfigService.setResourceService(ObjectBuilder.buildResourceService());
        Map<String, Object> data = mcpConfigService.config("classpath:mcp/mcp_client.json");
        Assert.assertNotNull(mcpConfigService.getResourceService());
        Assert.assertEquals("{\"mcpExports\":[\"mcp_server_tools_call@cr\"],\"mcpServers\":{\"gitlab\":{\"command\":\"npx\",\"args\":[\"-y\",\"@modelcontextprotocol/server-gitlab\"],\"env\":{\"GITLAB_PERSONAL_ACCESS_TOKEN\":\"glpat-MOCK_TOKEN_PLACEHOLDER\",\"GITLAB_API_URL\":\"https://git.com/api/v4\"}},\"SQLite Explorer\":{\"processor\":2,\"command\":\"python3\",\"args\":[\"src/test/resources/mcp/sqllite_server.py\"]},\"secure-filesystem-server\":{\"command\":\"npx\",\"args\":[\"-y\",\"@modelcontextprotocol/server-filesystem@2025.1.14\",\"/\",\"/\"]},\"echo\":{\"command\":\"/Library/Frameworks/Python.framework/Versions/3.12/bin/python3\",\"args\":[\"src/test/resources/mcp/echo_script.py\"]},\"test\":{\"command\":\"/Library/Frameworks/Python.framework/Versions/3.12/bin/python3\",\"args\":[\"src/test/resources/mcp/test_script.py\"]},\"empty-server\":{\"processor\":2,\"command\":\"python3\",\"args\":[\"src/test/resources/mcp/empty_server.py\"]},\"schema-server\":{\"processor\":2,\"command\":\"python3\",\"args\":[\"src/test/resources/mcp/schema_script.py\"]},\"multi_roles_prompt\":{\"processor\":2,\"command\":\"python3\",\"args\":[\"src/test/resources/mcp/multi_roles_prompt_script.py\"]},\"dynamic_resource\":{\"processor\":2,\"command\":\"python3\",\"args\":[\"src/test/resources/mcp/dynamic_resource.py\"]},\"http_streaming\":{\"type\":\"streamableHttp\",\"baseUrl\":\"http://127.0.0.1:8000/mcp\",\"headers\":{}},\"http_streaming_error\":{\"type\":\"streamableHttpError\",\"baseUrl\":\"http://127.0.0.1:8000/mcp\",\"headers\":{}}}}", JsonUtils.write(data));
        EasyMock.verify(placeholderResolver);
    }

    @Test
    public void testInit() throws Exception {
        String config = IOUtils.toString(new BufferedInputStream(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream()), StandardCharsets.UTF_8);
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.expect(placeholderResolver.replace(config)).andReturn(config).anyTimes();
        McpConfigInit init = EasyMock.createMock(McpConfigInit.class);
        init.init(EasyMock.anyObject(Map.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(init, placeholderResolver);
        McpConfigServiceImpl mcpConfigService = new McpConfigServiceImpl();
        mcpConfigService.setResourceService(ObjectBuilder.buildResourceService());
        mcpConfigService.setMcpConfigInit(Arrays.asList(init));
        mcpConfigService.setUri("classpath:mcp/mcp_client.json");
        mcpConfigService.setPlaceholderResolver(placeholderResolver);
        mcpConfigService.init();
        EasyMock.verify(init, placeholderResolver);
    }
}
