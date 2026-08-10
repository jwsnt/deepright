package ai.open.right.trace;

import ai.open.right.trace.impl.TraceServiceImpl;
import org.junit.Assert;
import org.junit.Test;

public class TraceServiceImplTest {

    @Test
    public void testCreate() {
        TraceServiceImpl service = new TraceServiceImpl();
        Assert.assertEquals(service.getTrace(null).length(),36);
    }

    @Test
    public void testReturn() {
        TraceServiceImpl service = new TraceServiceImpl();
        Assert.assertEquals(service.getTrace("Hello"),"Hello");
    }

    @org.junit.jupiter.api.Test
    public void testEmptyTrace() {
        TraceServiceImpl service = new TraceServiceImpl();
        org.junit.jupiter.api.Assertions.assertEquals(36, service.getTrace("").length());
    }
}
