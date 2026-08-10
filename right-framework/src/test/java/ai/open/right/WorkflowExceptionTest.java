package ai.open.right;

import ai.open.right.protocol.ProtocolCode;
import com.fasterxml.jackson.core.JsonParseException;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeoutException;

public class WorkflowExceptionTest {

    // ---------- exposure(Exception e) 及沿 cause 链解析的覆盖 ----------

    @Test
    public void testExposure_noWorkflowExceptionInChain_returnsFalse() {
        Assertions.assertFalse(WorkflowException.exposure(new Exception()));
        Assertions.assertFalse(WorkflowException.exposure(new RuntimeException("run")));
        Assertions.assertFalse(WorkflowException.exposure(new ExecutionException(new RuntimeException())));
    }

    @Test
    public void testExposure_directWorkflowException_exposureFlagRespected() {
        WorkflowException exposedEx = new WorkflowException("shown").needExposure();
        WorkflowException nonExposedEx = new WorkflowException("hidden", 500);
        Assertions.assertTrue(WorkflowException.exposure(exposedEx));
        Assertions.assertFalse(WorkflowException.exposure(nonExposedEx));
    }

    @Test
    public void testExposure_wrappedExposedWorkflowException_returnsTrue() {
        WorkflowException inner = new WorkflowException("client-visible").needExposure();
        Exception wrapped = new ExecutionException(inner);
        Assertions.assertTrue(WorkflowException.exposure(wrapped));
    }

    @Test
    public void testExposure_wrappedNonExposedWorkflowException_returnsFalse() {
        WorkflowException inner = new WorkflowException("internal", 500);
        Exception wrapped = new ExecutionException(inner);
        Assertions.assertFalse(WorkflowException.exposure(wrapped));
    }

    @Test
    public void testExposure_chainWithExposedWorkflowExceptionInCause_returnsTrue() {
        WorkflowException inner = new WorkflowException("exposed").needExposure();
        Exception chain = new Exception("outer", new RuntimeException("mid", inner));
        Assertions.assertTrue(WorkflowException.exposure(chain));
    }

    @Test
    public void testExposure_chainWithNoWorkflowException_returnsFalse() {
        Exception chain = new Exception("outer", new RuntimeException("mid", new IllegalArgumentException("cause")));
        Assertions.assertFalse(WorkflowException.exposure(chain));
    }

    // ---------- needExposure(Exception e) 覆盖 ----------

    @Test
    public void testNeedExposure_withExposedException_setsExposureTrueAndReturnsThis() {
        WorkflowException exposedCause = new WorkflowException("shown").needExposure();
        WorkflowException ex = new WorkflowException("wrap");
        WorkflowException result = ex.needExposure(exposedCause);
        Assertions.assertSame(ex, result);
        Assertions.assertTrue(ex.getExposure());
    }

    @Test
    public void testNeedExposure_withNonExposedException_setsExposureFalseAndReturnsThis() {
        Exception plain = new Exception("plain");
        WorkflowException ex = new WorkflowException("wrap");
        WorkflowException result = ex.needExposure(plain);
        Assertions.assertSame(ex, result);
        Assertions.assertFalse(ex.getExposure());
    }

    @Test
    public void testNeedExposure_withWrappedExposedWorkflowException_setsExposureTrue() {
        WorkflowException inner = new WorkflowException("client").needExposure();
        Exception wrapped = new ExecutionException(inner);
        WorkflowException ex = new WorkflowException("notify");
        ex.needExposure(wrapped);
        Assertions.assertTrue(ex.getExposure());
    }

    @Test
    public void testNeedExposure_withWrappedNonExposedWorkflowException_setsExposureFalse() {
        WorkflowException inner = new WorkflowException("internal", 500);
        Exception wrapped = new ExecutionException(inner);
        WorkflowException ex = new WorkflowException("notify");
        ex.needExposure(wrapped);
        Assertions.assertFalse(ex.getExposure());
    }

    // ---------- needExposure() 无参 ----------

    @Test
    public void testNeedExposure_noArg_setsTrueAndReturnsThis() {
        WorkflowException ex = new WorkflowException("x", 400);
        Assertions.assertFalse(ex.getExposure());
        WorkflowException result = ex.needExposure();
        Assertions.assertSame(ex, result);
        Assertions.assertTrue(ex.getExposure());
    }

    @Test
    public void testExposure_defaultFalseOnNewInstance() {
        WorkflowException ex = new WorkflowException("msg");
        Assertions.assertFalse(ex.getExposure());
    }

    // ---------- silent(Exception e) 及沿 cause 链解析的覆盖 ----------

    @Test
    public void testSilent_noWorkflowExceptionInChain_returnsFalse() {
        Assertions.assertFalse(WorkflowException.silent(new Exception()));
        Assertions.assertFalse(WorkflowException.silent(new RuntimeException("run")));
        Assertions.assertFalse(WorkflowException.silent(new ExecutionException(new RuntimeException())));
    }

    @Test
    public void testSilent_directWorkflowException_silentFlagRespected() {
        WorkflowException silentEx = new WorkflowException("closed").needSilent();
        WorkflowException nonSilentEx = new WorkflowException("error", 500);
        Assertions.assertTrue(WorkflowException.silent(silentEx));
        Assertions.assertFalse(WorkflowException.silent(nonSilentEx));
    }

    @Test
    public void testSilent_wrappedSilentWorkflowException_returnsTrue() {
        WorkflowException inner = new WorkflowException("task closed").needSilent();
        Exception wrapped = new ExecutionException(inner);
        Assertions.assertTrue(WorkflowException.silent(wrapped));
    }

    @Test
    public void testSilent_wrappedNonSilentWorkflowException_returnsFalse() {
        WorkflowException inner = new WorkflowException("error", 500);
        Exception wrapped = new ExecutionException(inner);
        Assertions.assertFalse(WorkflowException.silent(wrapped));
    }

    @Test
    public void testSilent_chainWithSilentWorkflowExceptionInCause_returnsTrue() {
        WorkflowException inner = new WorkflowException("closed").needSilent();
        Exception chain = new Exception("outer", new RuntimeException("mid", inner));
        Assertions.assertTrue(WorkflowException.silent(chain));
    }

    @Test
    public void testSilent_chainWithNoWorkflowException_returnsFalse() {
        Exception chain = new Exception("outer", new RuntimeException("mid", new IllegalArgumentException("cause")));
        Assertions.assertFalse(WorkflowException.silent(chain));
    }

    // ---------- needSilent(Exception e) 覆盖 ----------

    @Test
    public void testNeedSilent_withSilentException_setsSilentTrueAndReturnsThis() {
        WorkflowException silentCause = new WorkflowException("closed").needSilent();
        WorkflowException ex = new WorkflowException("wrap");
        WorkflowException result = ex.needSilent(silentCause);
        Assertions.assertSame(ex, result);
        Assertions.assertTrue(ex.getSilent());
    }

    @Test
    public void testNeedSilent_withNonSilentException_setsSilentFalseAndReturnsThis() {
        Exception plain = new Exception("plain");
        WorkflowException ex = new WorkflowException("wrap");
        WorkflowException result = ex.needSilent(plain);
        Assertions.assertSame(ex, result);
        Assertions.assertFalse(ex.getSilent());
    }

    @Test
    public void testNeedSilent_withWrappedSilentWorkflowException_setsSilentTrue() {
        WorkflowException inner = new WorkflowException("task closed").needSilent();
        Exception wrapped = new ExecutionException(inner);
        WorkflowException ex = new WorkflowException("notify");
        ex.needSilent(wrapped);
        Assertions.assertTrue(ex.getSilent());
    }

    @Test
    public void testNeedSilent_withWrappedNonSilentWorkflowException_setsSilentFalse() {
        WorkflowException inner = new WorkflowException("error", 500);
        Exception wrapped = new ExecutionException(inner);
        WorkflowException ex = new WorkflowException("notify");
        ex.needSilent(wrapped);
        Assertions.assertFalse(ex.getSilent());
    }

    @Test
    public void test() {
        WorkflowException e = new WorkflowException("test");
        WorkflowException code = new WorkflowException("test", 202);
        WorkflowException exception1 = new WorkflowException(new Exception());
        WorkflowException exception2 = new WorkflowException(new IllegalArgumentException());
        Assertions.assertFalse(e.getRetry());
        e.needSilent();
        Assertions.assertTrue(e.getSilent());
        Assertions.assertFalse(WorkflowException.retry(new Exception()));
        Assertions.assertFalse(WorkflowException.retry(e));
        Assertions.assertFalse(WorkflowException.retry(code));
        Assertions.assertFalse(WorkflowException.retry(exception1));
        Assertions.assertFalse(WorkflowException.retry(exception2));
        Assertions.assertFalse(WorkflowException.silent(new Exception()));
        Assertions.assertTrue(WorkflowException.silent(e));
        Assertions.assertFalse(WorkflowException.silent(code));
        Assertions.assertFalse(WorkflowException.silent(exception1));
        Assertions.assertFalse(WorkflowException.silent(exception2));
        Assertions.assertEquals(500, WorkflowException.code(new Exception()));
        Assertions.assertEquals(500, WorkflowException.code(e));
        Assertions.assertEquals(202, WorkflowException.code(code));
        Assertions.assertEquals(500, WorkflowException.code(exception1));
        Assertions.assertEquals(400, WorkflowException.code(exception2));
        WorkflowException workflowException = new WorkflowException();
        Assertions.assertEquals(workflowException, WorkflowException.create(workflowException, 200));
        Assertions.assertEquals(WorkflowException.class, WorkflowException.create(new RuntimeException(), 200).getClass());
    }

    @Test
    public void testCreate() {
        WorkflowException workflowException = new WorkflowException(new RuntimeException("OK"), 999);
        Assertions.assertEquals(999, workflowException.getCode());
        Assertions.assertEquals("java.lang.RuntimeException: OK", workflowException.getMessage());
    }

    @Test
    public void testCause() {
        RuntimeException runtimeException = new RuntimeException(new RuntimeException(new RuntimeException("OK")));
        Assertions.assertEquals("OK", WorkflowException.create(runtimeException, ProtocolCode.C500).getMessage());
    }

    @Test
    public void testCode() {
        Assertions.assertEquals(123, WorkflowException.code(new WorkflowException("OK", 123)));
        Assertions.assertEquals(400, WorkflowException.code(new IllegalArgumentException()));
        Assertions.assertEquals(400, WorkflowException.code(new JsonParseException(null, "OK")));
        Assertions.assertEquals(502, WorkflowException.code(new TimeoutException()));
        Assertions.assertEquals(503, WorkflowException.code(new IOException()));
        Assertions.assertEquals(500, WorkflowException.code(new RuntimeException()));
        Assertions.assertEquals(503, WorkflowException.code(new ExecutionException(new RuntimeException())));
    }

    @Test
    public void testConstructorWithCauseAndCode() {
        Exception cause = new RuntimeException("cause");
        WorkflowException ex = new WorkflowException(cause, 500);
        Assertions.assertEquals(cause, ex.getCause());
    }

    @Test
    public void testAdditionalCoverage() {
        Exception simpleEx = new Exception("simple");
        WorkflowException wEx1 = new WorkflowException(simpleEx, 501);
        Assertions.assertEquals(simpleEx, wEx1.getCause());
        Assertions.assertEquals(501, wEx1.getCode());

        WorkflowException wEx2 = new WorkflowException(new RuntimeException("run"));
        Assertions.assertEquals(500, wEx2.getCode());

        WorkflowException wEx3 = WorkflowException.create(new java.util.concurrent.TimeoutException("timeout"));
        Assertions.assertEquals(502, wEx3.getCode());

        WorkflowException wEx4 = new WorkflowException("test");
        Assertions.assertFalse(WorkflowException.retry(wEx4));
        Assertions.assertFalse(WorkflowException.silent(wEx4));
    }

    @Test
    public void testCoverageEnhancement() {
        WorkflowException exDefault = new WorkflowException();
        Assertions.assertEquals(ProtocolCode.C500, exDefault.getCode());

        Exception noCauseEx = new Exception("no cause");
        WorkflowException exNoCause = new WorkflowException(noCauseEx, 404);
        Assertions.assertEquals(noCauseEx, exNoCause.getCause());

        WorkflowException exCreateMsg = WorkflowException.create("msg");
        Assertions.assertEquals("msg", exCreateMsg.getMessage());
        Assertions.assertEquals(ProtocolCode.C500, exCreateMsg.getCode());

        WorkflowException exCreateMsgCode = WorkflowException.create("msg", 201);
        Assertions.assertEquals("msg", exCreateMsgCode.getMessage());
        Assertions.assertEquals(201, exCreateMsgCode.getCode());

        WorkflowException original = new WorkflowException("orig", 100);
        WorkflowException wrapped = WorkflowException.create(original, 200);
        Assertions.assertSame(original, wrapped);

        Exception eNoCause = new Exception("e no cause");
        WorkflowException exCreatedNoCause = WorkflowException.create(eNoCause, 300);
        Assertions.assertEquals("e no cause", exCreatedNoCause.getMessage());

        WorkflowException exNullCode = new WorkflowException("null code", null);
        Assertions.assertEquals(ProtocolCode.C500, WorkflowException.code(exNullCode));

        Assertions.assertFalse(WorkflowException.retry(new RuntimeException()));
        Assertions.assertFalse(WorkflowException.silent(new RuntimeException()));

        Assertions.assertEquals(ProtocolCode.C400, WorkflowException.code(new IllegalArgumentException()));
        Assertions.assertEquals(ProtocolCode.C503, WorkflowException.code(new ExecutionException(new Exception())));
        Assertions.assertEquals(ProtocolCode.C400, WorkflowException.code(new JsonParseException(null, "json")));
        Assertions.assertEquals(ProtocolCode.C502, WorkflowException.code(new TimeoutException()));
        Assertions.assertEquals(ProtocolCode.C503, WorkflowException.code(new IOException()));
        Assertions.assertEquals(ProtocolCode.C500, WorkflowException.code(new RuntimeException()));
    }

    @Test
    public void testCheck_withCode_whenConditionTrue_throwsWorkflowExceptionWithGivenCode() {
        WorkflowException ex = Assertions.assertThrows(WorkflowException.class,
                () -> WorkflowException.checkCondition(true, "boom", 418));
        Assertions.assertEquals("boom", ex.getMessage());
        Assertions.assertEquals(418, ex.getCode());
    }

    @Test
    public void testCheck_withCode_whenConditionFalse_doesNotThrow() {
        Assertions.assertDoesNotThrow(() -> WorkflowException.checkCondition(false, "boom", 418));
    }

    @Test
    public void testCheck_withExposure_whenConditionTrue_throwsExposedWorkflowException() {
        WorkflowException ex = Assertions.assertThrows(WorkflowException.class,
                () -> WorkflowException.checkCondition(true, true, "client-visible", 418));
        Assertions.assertEquals("client-visible", ex.getMessage());
        Assertions.assertEquals(418, ex.getCode());
        Assertions.assertTrue(ex.getExposure());
    }

    @Test
    public void testCheck_withExposureAndSilent_whenConditionTrue_throwsConfiguredWorkflowException() {
        WorkflowException ex = Assertions.assertThrows(WorkflowException.class,
                () -> WorkflowException.checkCondition(true, true, true, "client-visible", 418));
        Assertions.assertEquals("client-visible", ex.getMessage());
        Assertions.assertEquals(418, ex.getCode());
        Assertions.assertTrue(ex.getExposure());
        Assertions.assertTrue(ex.getSilent());
    }

    @Test
    public void testCheck_withoutExposure_whenConditionTrue_throwsNonExposedWorkflowException() {
        WorkflowException ex = Assertions.assertThrows(WorkflowException.class,
                () -> WorkflowException.checkCondition(true, false, "internal", 500));
        Assertions.assertEquals("internal", ex.getMessage());
        Assertions.assertEquals(500, ex.getCode());
        Assertions.assertFalse(ex.getExposure());
    }

    @Test
    public void testCheck_withExposure_whenConditionFalse_doesNotThrow() {
        Assertions.assertDoesNotThrow(() -> WorkflowException.checkCondition(false, true, "ignored", 418));
    }

    @Test
    public void testCheck_withoutCode_whenConditionTrue_throwsWorkflowExceptionWithDefaultCode() {
        WorkflowException ex = Assertions.assertThrows(WorkflowException.class,
                () -> WorkflowException.checkCondition(true, "boom"));
        Assertions.assertEquals("boom", ex.getMessage());
        Assertions.assertEquals(ProtocolCode.C500, ex.getCode());
    }

    @Test
    public void testCheck_withoutCode_whenConditionFalse_doesNotThrow() {
        Assertions.assertDoesNotThrow(() -> WorkflowException.checkCondition(false, "boom"));
    }

    @Test
    public void testCheckSilent_whenConditionTrue_throwsWorkflowExceptionWithSilentCode() {
        WorkflowException ex = Assertions.assertThrows(WorkflowException.class,
                () -> WorkflowException.checkSilent(true, "boom"));
        Assertions.assertEquals("boom", ex.getMessage());
        Assertions.assertEquals(ProtocolCode.C915, ex.getCode());
    }

    @Test
    public void testCheckSilent_whenConditionFalse_doesNotThrow() {
        Assertions.assertDoesNotThrow(() -> WorkflowException.checkSilent(false, "boom"));
    }
}
