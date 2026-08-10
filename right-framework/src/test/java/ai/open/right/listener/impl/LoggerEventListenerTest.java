package ai.open.right.listener.impl;

import ai.open.right.ObjectBuilder;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

public class LoggerEventListenerTest {

    @Test
    public void test() throws Exception {
        LoggerEventListener loggerListener = new LoggerEventListener();
        loggerListener.listen(ObjectBuilder.buildEvent());
    }

    @org.junit.jupiter.api.Test
    public void testListenNullEvent() throws Exception {
        LoggerEventListener loggerListener = new LoggerEventListener();
        // 验证 listen(null) 抛出 NullPointerException
        Assertions.assertThrows(NullPointerException.class, () -> {
            loggerListener.listen(null);
        });
    }

    @org.junit.jupiter.api.Test
    public void testListenEmptyEvent() throws Exception {
        LoggerEventListener loggerListener = new LoggerEventListener();
        // 验证 listen(empty event) 不抛出异常
        loggerListener.listen(new ai.open.right.listener.EventImpl());
    }

    @org.junit.jupiter.api.Test
    public void testListenEventWithNullData() throws Exception {
        LoggerEventListener loggerListener = new LoggerEventListener();
        ai.open.right.listener.EventImpl event = new ai.open.right.listener.EventImpl();
        // 显式设置 data 为 null
        event.setData(null);
        // 验证当 event 的 data 为 null 时，listen 方法能够正常处理而不抛出异常
        loggerListener.listen(event);
    }
}

