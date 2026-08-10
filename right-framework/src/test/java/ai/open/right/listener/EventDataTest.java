package ai.open.right.listener;

import ai.open.right.ObjectBuilder;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.Map;

public class EventDataTest {

    @Test
    public void testInit() {
        EventImpl eventData = new EventImpl(ObjectBuilder.buildEvent());
        Assert.assertEquals("DATA", eventData.getData());
        Assert.assertEquals("BIZ", eventData.getBiz());
        Assert.assertEquals("CHAT", eventData.getChat());
        Assert.assertEquals(Long.valueOf(10086L), eventData.getNow());
        Assert.assertEquals("TYPE", eventData.getType());
        Assert.assertEquals("DEVICE", eventData.getDevice());
    }

    @Test
    public void testSetGet() {
        EventImpl eventData = new EventImpl();
        eventData.setData("HELLO");
        eventData.setBiz("BIZ");
        eventData.setChat("CHAT");
        eventData.setNow(10086L);
        eventData.setType("TYPE");
        eventData.setDevice("DEVICE");
        Assert.assertEquals("HELLO", eventData.getData());
        Assert.assertEquals("BIZ", eventData.getBiz());
        Assert.assertEquals("CHAT", eventData.getChat());
        Assert.assertEquals(Long.valueOf(10086L), eventData.getNow());
        Assert.assertEquals("TYPE", eventData.getType());
        Assert.assertEquals("DEVICE", eventData.getDevice());
    }

    @Test
    public void testConvert() {
        EventImpl eventData = new EventImpl();
        eventData.setData(Collections.singletonMap("KEY", "VAL"));
        eventData.setBiz("BIZ");
        eventData.setChat("CHAT");
        eventData.setNow(10086L);
        eventData.setType("TYPE");
        eventData.setDevice("DEVICE");
        Assert.assertEquals("VAL", Map.class.cast(eventData.getData()).get("KEY"));
        Assert.assertEquals("BIZ", eventData.getBiz());
        Assert.assertEquals("CHAT", eventData.getChat());
        Assert.assertEquals(Long.valueOf(10086L), eventData.getNow());
        Assert.assertEquals("TYPE", eventData.getType());
        Assert.assertEquals("DEVICE", eventData.getDevice());
    }
}
