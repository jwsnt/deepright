package ai.open.right.listener;

import ai.open.right.ObjectBuilder;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.listener.impl.LoggerEventListener;
import org.easymock.EasyMock;
import org.junit.Test;

import java.util.Arrays;

public class EventListenerTest {

    @Test
    public void test() throws Exception {
        LoggerEventListener eventLogListener = new LoggerEventListener();
        EventListenerServiceImpl listenerManager = new EventListenerServiceImpl();
        listenerManager.setEventListener(Arrays.asList(eventLogListener));
        listenerManager.listen(ObjectBuilder.buildEvent());
    }

    @Test
    public void testWithException() throws Exception {
        Event event = ObjectBuilder.buildEvent();
        EventListener eventListener = EasyMock.createMock(EventListener.class);
        eventListener.listen(event);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(eventListener);
        EventListenerServiceImpl listenerManager = new EventListenerServiceImpl();
        listenerManager.setEventListener(Arrays.asList(eventListener));
        listenerManager.listen(event);
        EasyMock.verify(eventListener);
    }

    @Test
    public void testAnonymousListener() throws Exception {
        final boolean[] called = {false};
        EventListener listener = new EventListener() {
            @Override
            public void listen(Event event) throws Exception {
                called[0] = true;
            }
        };
        listener.listen(ObjectBuilder.buildEvent());
        org.junit.Assert.assertTrue(called[0]);
    }

    @org.junit.jupiter.api.Test
    public void testMultipleListeners() throws Exception {
        final int[] callCount = {0};
        EventListener l1 = event -> callCount[0]++;
        EventListener l2 = event -> callCount[0]++;
        EventListenerServiceImpl service = new EventListenerServiceImpl();
        service.setEventListener(java.util.Arrays.asList(l1, l2));
        ai.open.right.listener.EventImpl event = new ai.open.right.listener.EventImpl();
        event.setData("test");
        service.listen(event);
        org.junit.jupiter.api.Assertions.assertEquals(2, callCount[0]);
    }

}
