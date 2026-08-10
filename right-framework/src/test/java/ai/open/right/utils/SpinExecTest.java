package ai.open.right.utils;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * SpinExec 单元测试类
 * 覆盖率目标：99%+
 */
class SpinExecTest {

    /**
     * Mock 实现类，用于控制 doExec 的行为
     */
    private static class MockSpinExec extends SpinExec {
        private final long doExecSleepMs;
        private final Object result;
        private final int successAtCircle;
        private final boolean throwException;
        private final AtomicInteger currentCircle = new AtomicInteger(0);

        public MockSpinExec(Integer timeout, Integer circle, long doExecSleepMs, Object result, int successAtCircle, boolean throwException) {
            super(timeout, circle);
            this.doExecSleepMs = doExecSleepMs;
            this.result = result;
            this.successAtCircle = successAtCircle;
            this.throwException = throwException;
        }

        @Override
        public Object doExec() throws Exception {
            int count = currentCircle.incrementAndGet();
            if (doExecSleepMs > 0) {
                Thread.sleep(doExecSleepMs);
            }
            if (throwException) {
                throw new RuntimeException("Mock exception");
            }
            if (count == successAtCircle) {
                return result;
            }
            return null;
        }

        public int getActualCallCount() {
            return currentCircle.get();
        }
    }

    @Test
    void testCircleZero() throws Exception {
        // 测试 circle <= 0 的情况
        MockSpinExec exec = new MockSpinExec(100, 0, 0, "OK", 1, false);
        Assertions.assertNull(exec.exec(), "当 circle <= 0 时应直接返回 null");
    }

    @Test
    void testSuccessFirstTime() throws Exception {
        // 测试第一次执行就成功返回
        MockSpinExec exec = new MockSpinExec(1000, 5, 0, "OK", 1, false);
        Assertions.assertEquals("OK", exec.exec());
        Assertions.assertEquals(1, exec.getActualCallCount());
    }

    @Test
    void testSuccessAfterRetries() throws Exception {
        // 测试经过多次循环后成功返回，且包含睡眠逻辑
        MockSpinExec exec = new MockSpinExec(1000, 5, 10, "OK", 3, false);
        Assertions.assertEquals("OK", exec.exec());
        Assertions.assertEquals(3, exec.getActualCallCount());
    }

    @Test
    void testException() {
        // 测试 doExec 抛出异常的情况
        MockSpinExec exec = new MockSpinExec(1000, 5, 0, "OK", 1, true);
        Assertions.assertThrows(RuntimeException.class, exec::exec);
    }

    @Test
    void testSkipSleep() throws Exception {
        // 测试执行时间过长，跳过睡眠的逻辑
        MockSpinExec exec = new MockSpinExec(50, 2, 30, "OK", 2, false);
        Assertions.assertEquals("OK", exec.exec());
    }

    @Test
    void testTotalQuotaReached() throws Exception {
        // 模拟执行时间超过总配额，触发 break
        MockSpinExec exec = new MockSpinExec(100, 5, 150, null, 99, false);
        Object res = exec.exec();
        Assertions.assertNull(res, "超时后应返回 null");
    }

    @Test
    void testLoopFinishedWithNull() throws Exception {
        // 模拟所有循环执行完毕且未返回结果，最终返回 null
        MockSpinExec exec = new MockSpinExec(500, 2, 0, "DATA", 99, false);
        Object res = exec.exec();
        Assertions.assertNull(res, "循环结束未命中应返回 null");
        Assertions.assertEquals(2, exec.getActualCallCount());
    }

    @Test
    void testMultipleCyclesNoResult() throws Exception {
        MockSpinExec exec = new MockSpinExec(500, 3, 10, "DATA", 99, false);
        Object res = exec.exec();
        Assertions.assertNull(res, "多轮循环无结果应返回 null");
        Assertions.assertEquals(3, exec.getActualCallCount());
    }

    @Test
    void testNanosSleep() throws Exception {
        MockSpinExec exec = new MockSpinExec(100, 1, 0, null, 99, false);
        exec.exec();
    }

    @Test
    void testGetters() {
        SpinExec exec = new MockSpinExec(123, 456, 0, null, 1, false);
        Assertions.assertEquals(123, exec.getTimeout());
        Assertions.assertEquals(456, exec.getCircle());
    }

    @Test
    void testInterruptedException() {
        MockSpinExec exec = new MockSpinExec(1000, 5, 0, null, 99, false) {
            @Override
            public Object doExec() throws Exception {
                Thread.currentThread().interrupt();
                return null;
            }
        };
        Assertions.assertThrows(InterruptedException.class, exec::exec);
    }
}
