package ai.open.right.trace;

import ai.open.right.trace.impl.TraceServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class TraceServiceImplInitConfigTest {

    @Test
    public void shouldCreateTraceServiceWithDefaults() throws Exception {
        TraceServiceImpl.InitConfig init = new TraceServiceImpl.InitConfig();
        TraceServiceImpl bean = (TraceServiceImpl) init.traceService();
        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof TraceService);
    }

    @Test
    public void shouldCreateTraceServiceInstance() throws Exception {
        TraceServiceImpl.InitConfig init = new TraceServiceImpl.InitConfig();
        TraceServiceImpl bean1 = (TraceServiceImpl) init.traceService();
        TraceServiceImpl bean2 = (TraceServiceImpl) init.traceService();
        // 每次调用都应该创建新的实例
        Assert.assertNotSame(bean1, bean2);
    }
}
