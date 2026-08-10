package ai.open.right.config;

import ai.open.right.WorkflowException;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;
import org.springframework.context.ApplicationContext;
import org.springframework.scheduling.annotation.AsyncAnnotationBeanPostProcessor;
import org.springframework.util.StringUtils;

import java.lang.Thread.UncaughtExceptionHandler;
import java.util.Collections;
import java.util.HashMap;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.atomic.AtomicReference;

public class ThreadsConfigTest {

    private static final Integer NUMBER = 10;

    @Test
    public void testClassic() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setShutdowninterval(ThreadsConfigTest.NUMBER);
        config.setKeepalive(ThreadsConfigTest.NUMBER);
        config.setQueue(ThreadsConfigTest.NUMBER);
        config.setCore(ThreadsConfigTest.NUMBER);
        config.setMax(ThreadsConfigTest.NUMBER);
        config.setMode(ThreadsConfig.MODE_CLASSIC);
        ExecutorService executor = config.executor();
        Assert.assertNull(config.getContext());
        ApplicationContext applicationContext = EasyMock.createMock(ApplicationContext.class);
        config.setContext(applicationContext);
        Assert.assertEquals(applicationContext, config.getContext());
        Assert.assertEquals(executor.getClass(), ThreadPoolExecutor.class);
        config.destroy();
        Assert.assertTrue(executor.isShutdown());
    }

    @Test
    public void testVirtual() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setShutdowninterval(ThreadsConfigTest.NUMBER);
        config.setInheritInheritableThreadLocals(false);
        config.setKeepalive(ThreadsConfigTest.NUMBER);
        config.setQueue(ThreadsConfigTest.NUMBER);
        config.setCore(ThreadsConfigTest.NUMBER);
        config.setMax(ThreadsConfigTest.NUMBER);
        config.setMode(ThreadsConfig.MODE_VIRTUAL);
        ExecutorService executor = config.executor();
        Assert.assertFalse(config.getInheritInheritableThreadLocals());
        Assert.assertEquals(executor.getClass().getName(), "java.util.concurrent.ThreadPerTaskExecutor");
        config.destroy();
        Assert.assertTrue(executor.isShutdown());
    }

    @Test
    public void testMonitor() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setShutdowninterval(ThreadsConfigTest.NUMBER);
        config.setKeepalive(ThreadsConfigTest.NUMBER);
        config.setQueue(ThreadsConfigTest.NUMBER);
        config.setCore(ThreadsConfigTest.NUMBER);
        config.setMax(ThreadsConfigTest.NUMBER);
        config.setMode(ThreadsConfig.MODE_VIRTUAL);
        config.destroy();
        String monitor = config.monitor();
        Assert.assertTrue(StringUtils.hasText(monitor));
    }

    @Test
    public void testMonitorDetailed() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setShutdowninterval(ThreadsConfigTest.NUMBER);
        config.setKeepalive(ThreadsConfigTest.NUMBER);
        config.setQueue(ThreadsConfigTest.NUMBER);
        config.setCore(ThreadsConfigTest.NUMBER);
        config.setMax(ThreadsConfigTest.NUMBER);
        config.setMode(ThreadsConfig.MODE_VIRTUAL);
        String monitor = config.monitor();
        // 验证 monitor 返回的字符串是否包含 "Exec: " 和 "Sys: "
        Assert.assertTrue(monitor.contains("Exec: "));
        Assert.assertTrue(monitor.contains("Sys: "));
    }

    // 新增 JUnit 5 测试方法
    @org.junit.jupiter.api.Test
    public void testMonitorDetailedJUnit5() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setShutdowninterval(ThreadsConfigTest.NUMBER);
        config.setKeepalive(ThreadsConfigTest.NUMBER);
        config.setQueue(ThreadsConfigTest.NUMBER);
        config.setCore(ThreadsConfigTest.NUMBER);
        config.setMax(ThreadsConfigTest.NUMBER);
        config.setMode(ThreadsConfig.MODE_VIRTUAL);
        String monitor = config.monitor();
        // 验证 monitor() 返回的字符串包含 "Exec: " 和 "Sys: "
        Assertions.assertTrue(monitor.contains("Exec: "), "Monitor string should contain 'Exec: '");
        Assertions.assertTrue(monitor.contains("Sys: "), "Monitor string should contain 'Sys: '");
    }

    @Test
    public void testInit() {
        AsyncAnnotationBeanPostProcessor asyncAnnotationBeanPostProcessor = EasyMock.createMock(AsyncAnnotationBeanPostProcessor.class);
        ApplicationContext applicationContext = EasyMock.createMock(ApplicationContext.class);
        EasyMock.expect(applicationContext.getBeansOfType(AsyncAnnotationBeanPostProcessor.class)).andReturn(Collections.singletonMap("Async", asyncAnnotationBeanPostProcessor)).anyTimes();
        EasyMock.replay(asyncAnnotationBeanPostProcessor, applicationContext);
        ThreadsConfig config = new ThreadsConfig();
        config.setContext(applicationContext);
        config.init();
        EasyMock.verify(asyncAnnotationBeanPostProcessor, applicationContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testInitWithNull() {
        ApplicationContext applicationContext = EasyMock.createMock(ApplicationContext.class);
        EasyMock.expect(applicationContext.getBeansOfType(AsyncAnnotationBeanPostProcessor.class)).andReturn(new HashMap<>()).anyTimes();
        EasyMock.replay(applicationContext);
        ThreadsConfig config = new ThreadsConfig();
        config.setContext(applicationContext);
        config.init();
        EasyMock.verify(applicationContext);
        Assert.assertNotNull(config.getContext());
        config.setContext(null);
        Assert.assertNull(config.getContext());
    }

    @org.junit.jupiter.api.Test
    public void testThreadsConfig() {
        ThreadsConfig config = new ThreadsConfig();
        org.junit.jupiter.api.Assertions.assertNotNull(config);
    }

    @org.junit.jupiter.api.Test
    public void testDestroyNullExecutor() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setExecutor(null);
        config.destroy();
    }

    // ---------- Classic 模式 withHandler：线程创建时 setUncaughtExceptionHandler(INSTANCE) 覆盖 ----------

    @org.junit.jupiter.api.Test
    public void classicExecutor_threadHasCustomizableUncaughtExceptionHandler() throws Exception {
        ThreadsConfig config = new ThreadsConfig();
        config.setShutdowninterval(ThreadsConfigTest.NUMBER);
        config.setKeepalive(ThreadsConfigTest.NUMBER);
        config.setQueue(ThreadsConfigTest.NUMBER);
        config.setCore(1);
        config.setMax(1);
        config.setMode(ThreadsConfig.MODE_CLASSIC);
        ExecutorService executor = config.executor();
        AtomicReference<UncaughtExceptionHandler> captured = new AtomicReference<>();
        CountDownLatch done = new CountDownLatch(1);
        executor.submit(() -> {
            captured.set(Thread.currentThread().getUncaughtExceptionHandler());
            done.countDown();
        });
        done.await();
        Assertions.assertSame(ThreadsConfig.CustomizableUncaughtExceptionHandler.INSTANCE, captured.get());
        config.destroy();
    }

    // ---------- CustomizableUncaughtExceptionHandler 覆盖 ----------

    @org.junit.jupiter.api.Test
    public void uncaughtException_workflowExceptionSilent_doesNotThrow() {
        Thread t = Thread.currentThread();
        WorkflowException e = new WorkflowException("The task was closed (main@main)").needSilent();
        ThreadsConfig.CustomizableUncaughtExceptionHandler.INSTANCE.uncaughtException(t, e);
    }

    @org.junit.jupiter.api.Test
    public void uncaughtException_workflowExceptionNotSilent_doesNotThrow() {
        Thread t = Thread.currentThread();
        WorkflowException e = new WorkflowException("error", 500);
        ThreadsConfig.CustomizableUncaughtExceptionHandler.INSTANCE.uncaughtException(t, e);
    }

    @org.junit.jupiter.api.Test
    public void uncaughtException_otherThrowable_doesNotThrow() {
        Thread t = Thread.currentThread();
        RuntimeException e = new RuntimeException("other");
        ThreadsConfig.CustomizableUncaughtExceptionHandler.INSTANCE.uncaughtException(t, e);
    }

    /**
     * 覆盖 else 分支：Throwable 非 Exception（如 Error）时走 log.error(e.getMessage(), e)，且不抛异常
     */
    @org.junit.jupiter.api.Test
    public void uncaughtException_whenThrowableIsError_invokesLogErrorAndDoesNotThrow() {
        Thread t = Thread.currentThread();
        AssertionError e = new AssertionError("assert failed");
        Assertions.assertDoesNotThrow(() ->
                ThreadsConfig.CustomizableUncaughtExceptionHandler.INSTANCE.uncaughtException(t, e));
    }

}
