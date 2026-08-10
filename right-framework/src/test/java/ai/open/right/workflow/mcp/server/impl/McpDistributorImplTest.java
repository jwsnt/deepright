package ai.open.right.workflow.mcp.server.impl;

import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpRequest;
import org.easymock.EasyMock;
import org.junit.Test;
import java.util.Collections;

public class McpDistributorImplTest {
    @Test
    public void testDistribute() throws Exception {
        McpDistributorImpl distributor = new McpDistributorImpl();
        McpCmdExportService service = EasyMock.createMock(McpCmdExportService.class);
        distributor.setCmdServices(Collections.singletonMap("method", service));
        
        McpRequest request = EasyMock.createMock(McpRequest.class);
        EasyMock.expect(request.getMethod()).andReturn("method").anyTimes();
        EasyMock.expect(request.getId()).andReturn("123").anyTimes();
        service.cmd(request);
        EasyMock.expectLastCall().once();
        
        EasyMock.replay(service, request);
        distributor.distribute(request);
        EasyMock.verify(service, request);
    }

    @Test
    public void testDistributeUnsupported() throws Exception {
        McpDistributorImpl distributor = new McpDistributorImpl();
        distributor.setCmdServices(Collections.emptyMap());
        McpRequest request = EasyMock.createMock(McpRequest.class);
        EasyMock.expect(request.getMethod()).andReturn("unsupported").anyTimes();
        EasyMock.expect(request.getId()).andReturn("1").anyTimes();
        request.error(EasyMock.anyString());
        EasyMock.expectLastCall().once();
        
        EasyMock.replay(request);
        distributor.distribute(request);
        EasyMock.verify(request);
    }
}
