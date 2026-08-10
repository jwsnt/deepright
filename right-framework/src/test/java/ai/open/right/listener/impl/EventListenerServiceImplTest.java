package ai.open.right.listener.impl;

import ai.open.right.listener.EventImpl;
import ai.open.right.listener.EventListener;
import org.junit.Assert;
import org.junit.Test;
import org.junit.jupiter.api.Assertions;

import java.util.ArrayList;
import java.util.List;

public class EventListenerServiceImplTest {

    @Test
    public void testSetGet() {
        List<EventListener> events = new ArrayList<EventListener>();
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        eventListenerService.setEventListener(events);
        Assert.assertEquals(events, eventListenerService.getEventListener());
    }

    @Test
    public void testGetDataException() throws Exception {
        List<EventListener> events = new ArrayList<EventListener>();
        events.add(new LoggerEventListener());
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        eventListenerService.setEventListener(events);
        eventListenerService.listen(new EventImpl());
    }

    @org.junit.jupiter.api.Test
    public void testListenEmptyListeners() throws Exception {
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        // Case 1: null list
        eventListenerService.setEventListener(null);
        eventListenerService.listen(new EventImpl());

        // Case 2: empty list
        eventListenerService.setEventListener(new ArrayList<EventListener>());
        eventListenerService.listen(new EventImpl());
    }

    @org.junit.jupiter.api.Test
    public void testListenWithExceptionInOneListener() throws Exception {
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        List<EventListener> listeners = new ArrayList<EventListener>();
        final boolean[] secondCalled = {false};
        listeners.add(new EventListener() {
            @Override
            public void listen(ai.open.right.listener.Event event) throws Exception {
                throw new RuntimeException("test exception");
            }
        });
        listeners.add(new EventListener() {
            @Override
            public void listen(ai.open.right.listener.Event event) throws Exception {
                secondCalled[0] = true;
            }
        });
        eventListenerService.setEventListener(listeners);
        EventImpl event = new EventImpl();
        event.setData("test data");
        eventListenerService.listen(event);
        Assertions.assertTrue(secondCalled[0], "Second listener should be called even if the first one throws an exception");
    }

    @org.junit.jupiter.api.Test
    public void testInitConfig() throws Exception {
        EventListenerServiceImpl.InitConfig config = new EventListenerServiceImpl.InitConfig();
        List<EventListener> listeners = new ArrayList<EventListener>();
        config.setEventListener(listeners);
        ai.open.right.listener.EventListenerService service = config.eventListenerService();
        Assertions.assertNotNull(service);
        Assertions.assertTrue(service instanceof EventListenerServiceImpl);
        Assertions.assertEquals(listeners, ((EventListenerServiceImpl) service).getEventListener());
    }

    @org.junit.jupiter.api.Test
    public void testListenValidEvent() throws Exception {
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        List<EventListener> listeners = new ArrayList<EventListener>();
        final int[] callCount = {0};
        listeners.add(new EventListener() {
            @Override
            public void listen(ai.open.right.listener.Event event) throws Exception {
                callCount[0]++;
            }
        });
        eventListenerService.setEventListener(listeners);
        EventImpl event = new EventImpl();
        event.setData("valid data");
        eventListenerService.listen(event);
        Assertions.assertEquals(1, callCount[0]);
    }

    @org.junit.jupiter.api.Test
    public void testListenNullEvent() throws Exception {
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        List<EventListener> listeners = new ArrayList<EventListener>();
        listeners.add(new EventListener() {
            @Override
            public void listen(ai.open.right.listener.Event event) throws Exception {
            }
        });
        eventListenerService.setEventListener(listeners);
        // Should not throw exception due to try-catch in listen method
        eventListenerService.listen(null);
    }

    @org.junit.jupiter.api.Test
    public void testListenWithNullData() throws Exception {
        EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
        List<EventListener> listeners = new ArrayList<>();
        listeners.add(new LoggerEventListener());
        eventListenerService.setEventListener(listeners);
        ai.open.right.listener.EventImpl event = new ai.open.right.listener.EventImpl();
        event.setData(null);
        // 内部捕获 Assert.notNull 抛出的异常，不应向外抛出
        eventListenerService.listen(event);
    }
}

