package ai.open.right.config;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import static org.junit.jupiter.api.Assertions.*;

@ExtendWith(MockitoExtension.class)
class ThreadsGetSetConfigTest {

    @InjectMocks
    private ThreadsConfig threadsConfig;

    @BeforeEach
    void setUp() {
        // 模拟注入@Value属性
        ReflectionTestUtils.setField(threadsConfig, "shutdowninterval", 2000);
        ReflectionTestUtils.setField(threadsConfig, "keepalive", 30000);
        ReflectionTestUtils.setField(threadsConfig, "queue", 200);
        ReflectionTestUtils.setField(threadsConfig, "core", 50);
        ReflectionTestUtils.setField(threadsConfig, "max", 150);
        ReflectionTestUtils.setField(threadsConfig, "mode", ThreadsConfig.MODE_CLASSIC);
    }

    @Test
    void testShutdowninterval() {
        assertEquals(2000, threadsConfig.getShutdowninterval());
        threadsConfig.setShutdowninterval(3000);
        assertEquals(3000, threadsConfig.getShutdowninterval());
    }

    @Test
    void testKeepalive() {
        assertEquals(30000, threadsConfig.getKeepalive());
        threadsConfig.setKeepalive(45000);
        assertEquals(45000, threadsConfig.getKeepalive());
    }

    @Test
    void testQueue() {
        assertEquals(200, threadsConfig.getQueue());
        threadsConfig.setQueue(300);
        assertEquals(300, threadsConfig.getQueue());
    }

    @Test
    void testCore() {
        assertEquals(50, threadsConfig.getCore());
        threadsConfig.setCore(75);
        assertEquals(75, threadsConfig.getCore());
    }

    @Test
    void testMax() {
        assertEquals(150, threadsConfig.getMax());
        threadsConfig.setMax(200);
        assertEquals(200, threadsConfig.getMax());
    }

    @Test
    void testMode() {
        assertEquals(ThreadsConfig.MODE_CLASSIC, threadsConfig.getMode());
        threadsConfig.setMode(ThreadsConfig.MODE_VIRTUAL);
        assertEquals(ThreadsConfig.MODE_VIRTUAL, threadsConfig.getMode());
        threadsConfig.setMode(ThreadsConfig.MODE_CLASSIC);
        assertEquals(ThreadsConfig.MODE_CLASSIC, threadsConfig.getMode());
    }

    @Test
    void testExecutor() {
        assertNull(threadsConfig.getExecutor());
        // 测试设置ExecutorService
        ExecutorService testExecutor = Executors.newSingleThreadExecutor();
        threadsConfig.setExecutor(testExecutor);
        assertSame(testExecutor, threadsConfig.getExecutor());
        // 清理
        testExecutor.shutdown();
    }
}
