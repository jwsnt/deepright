package ai.open.right.listener;

import ai.open.right.ObjectBuilder;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import org.easymock.EasyMock;
import org.junit.Test;

public class EventListenerManagerTest {

    @Test
    public void test() throws Exception {
        Event event = ObjectBuilder.buildEvent();
        EventListener eventListener = EasyMock.createMock(EventListener.class);
        eventListener.listen(event);
        EasyMock.expectLastCall().andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(eventListener);
        EventListenerServiceImpl eventListenerManager = new EventListenerServiceImpl();
        eventListenerManager.listen(event);
        EasyMock.verify(eventListener);
    }
}
