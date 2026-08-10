package ai.open.right.integration.impl;

import ai.open.right.trace.TraceService;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class RightServiceImplInitConfigTest {

    @Test
    public void shouldCreateRightServiceWithProvidedProperties() throws Exception {
        RightServiceImpl.InitConfig init = new RightServiceImpl.InitConfig();
        TraceService traceService = EasyMock.createMock(TraceService.class);
        Workflow workflow = EasyMock.createMock(Workflow.class);
        workflow.async(EasyMock.anyObject(WorkflowTask.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(traceService, workflow);
        // setter 注入
        init.setTraceService(traceService);
        init.setWorkflow(workflow);
        init.setTimeout(123);
        RightServiceImpl bean = (RightServiceImpl) init.rightService();
        // 使用 getter 验证
        Assert.assertSame(traceService, bean.getTraceService());
        Assert.assertSame(workflow, bean.getWorkflow());
        Assert.assertEquals(Integer.valueOf(123), bean.getTimeout());
        EasyMock.verify(traceService, workflow);
    }

    @Test
    public void shouldCreateRightServiceWithDefaultsWhenNoPropertiesProvided() throws Exception {
        RightServiceImpl.InitConfig init = new RightServiceImpl.InitConfig();
        RightServiceImpl bean = (RightServiceImpl) init.rightService();
        Assert.assertNull(bean.getTraceService());
        Assert.assertNull(bean.getWorkflow());
        Assert.assertNull(bean.getTimeout());
    }
}
