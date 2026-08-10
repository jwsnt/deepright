package ai.open.right.listener;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

/**
 * EventImpl 单元测试类
 */
public class EventImplTest {

    /**
     * 测试从另一个 Event 对象构造
     */
    @Test
    public void testConstructorWithEvent() {
        EventImpl original = new EventImpl();
        original.setWorkflow("workflow");
        original.setDevice("device1");
        original.setChat("chat1");
        original.setType("type1");
        original.setData("data1");
        original.setBiz("biz1");
        original.setNow(123456789L);

        EventImpl copy = new EventImpl(original);
        Assertions.assertEquals("workflow", copy.getWorkflow());
        Assertions.assertEquals("device1", copy.getDevice());
        Assertions.assertEquals("chat1", copy.getChat());
        Assertions.assertEquals("type1", copy.getType());
        Assertions.assertEquals("data1", copy.getData());
        Assertions.assertEquals("biz1", copy.getBiz());
        Assertions.assertEquals(Long.valueOf(123456789L), copy.getNow());
    }

    /**
     * 测试 getDimension 在不同 biz, chat, device 组合下的返回值
     */
    @Test
    public void testGetDimension() {
        EventImpl event = new EventImpl();
        event.setBiz("biz");
        event.setChat("chat");
        event.setDevice("device");
        Assertions.assertEquals("biz-chat-device", event.getDimension());

        event.setBiz(null);
        Assertions.assertEquals("-chat-device", event.getDimension());

        event.setChat(null);
        Assertions.assertEquals("--device", event.getDimension());

        event.setDevice(null);
        Assertions.assertEquals("--", event.getDimension());
    }

    /**
     * 测试 init() 方法
     */
    @Test
    public void testInit() {
        EventImpl event = new EventImpl();
        EventImpl initialized = event.init();
        Assertions.assertSame(event, initialized);
    }

    /**
     * 覆盖所有 setter 和 getter
     */
    @Test
    public void testSettersAndGetters() {
        EventImpl event = new EventImpl();
        
        event.setDevice("d");
        Assertions.assertEquals("d", event.getDevice());
        
        event.setType("t");
        Assertions.assertEquals("t", event.getType());
        
        Object data = new Object();
        event.setData(data);
        Assertions.assertEquals(data, event.getData());
        
        event.setChat("c");
        Assertions.assertEquals("c", event.getChat());
        
        event.setBiz("b");
        Assertions.assertEquals("b", event.getBiz());
        
        Long now = System.currentTimeMillis();
        event.setNow(now);
        Assertions.assertEquals(now, event.getNow());
    }

    /**
     * 测试默认构造函数
     */
    @Test
    public void testDefaultConstructor() {
        EventImpl event = new EventImpl();
        Assertions.assertNull(event.getDevice());
        Assertions.assertNull(event.getChat());
        Assertions.assertNull(event.getType());
        Assertions.assertNull(event.getData());
        Assertions.assertNull(event.getBiz());
        Assertions.assertNull(event.getNow());
    }

    @Test
    public void testGetDimensionEmpty() {
        EventImpl event = new EventImpl();
        event.setBiz("");
        event.setChat("");
        event.setDevice("");
        Assertions.assertEquals("--", event.getDimension());
    }

    @org.junit.jupiter.api.Test
    public void testConstructorNull() {
        org.junit.jupiter.api.Assertions.assertThrows(NullPointerException.class, () -> {
            new EventImpl(null);
        });
    }

}
