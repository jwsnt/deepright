package ai.open.right.listener.impl;

import java.util.Arrays;
import java.util.List;

import org.easymock.EasyMock;
import org.junit.Test;

import ai.open.right.listener.Event;
import ai.open.right.listener.EventListener;

public class EventListenerServiceImplInitConfigTest {

    @Test
    public void shouldCreateEventListenerServiceWithProvidedProperties() throws Exception {
        EventListenerServiceImpl.InitConfig init = new EventListenerServiceImpl.InitConfig();

        EventListener listener1 = EasyMock.createMock(EventListener.class);
        EventListener listener2 = EasyMock.createMock(EventListener.class);
        List<EventListener> listeners = Arrays.asList(listener1, listener2);
        // setter 注入
        init.setEventListener(listeners);
        EventListenerServiceImpl bean = (EventListenerServiceImpl) init.eventListenerService();
        // 使用反射或方法行为验证复制
        Event event = EasyMock.createMock(Event.class);
        EasyMock.expect(event.getData()).andReturn(new Object()).anyTimes();
        EasyMock.expect(event.init()).andReturn(event).anyTimes();

        listener1.listen(event);
        EasyMock.expectLastCall().once();
        listener2.listen(event);
        EasyMock.expectLastCall().once();

        EasyMock.replay(listener1, listener2, event);

        bean.listen(event);

        EasyMock.verify(listener1, listener2, event);
    }

    @Test
    public void shouldCreateEventListenerServiceWithDefaultsWhenNoPropertiesProvided() throws Exception {
        EventListenerServiceImpl.InitConfig init = new EventListenerServiceImpl.InitConfig();

        EventListenerServiceImpl bean = (EventListenerServiceImpl) init.eventListenerService();

        // 不抛异常即可
        bean.listen(new Event() {
            @Override
            public String getType() {
                return "t";
            }

            @Override
            public Object getData() {
                return new Object();
            }

            @Override
            public Long getNow() {
                return 0L;
            }

            @Override
            public Event init() {
                return this;
            }

            @Override
            public String getDimension() {
                return "biz-chat-device";
            }

            @Override
            public String getBiz() {
                return "biz";
            }

            @Override
            public String getChat() {
                return "chat";
            }

            @Override
            public String getDevice() {
                return "device";
            }

            @Override
            public String getWorkflow() {
                return "workflow";
            }
        });
    }
}

 