package ai.open.right.workflow.flow.pubsub;

import ai.open.right.protocol.ProtocolCode;
import org.junit.Assert;
import org.junit.Test;

public class PubSubConfigTest {

    @Test
    public void test() {
        PubSubConfig pubSubConfig = new PubSubConfig();
        Assert.assertFalse(pubSubConfig.getContainHistories());
        Assert.assertEquals("NOTIFIER", pubSubConfig.getNotifier("NOTIFIER"));
        Assert.assertEquals(Integer.valueOf(1000), pubSubConfig.getTimeout4Llm(1000));
        pubSubConfig.setFormatter("FORMAT");
        pubSubConfig.setNotifier("NOTIFIER");
        pubSubConfig.setReply("REPLAY");
        pubSubConfig.setTimeout4Llm(2000);
        pubSubConfig.setContainHistories(true);
        Assert.assertTrue(pubSubConfig.getContainHistories());
        Assert.assertEquals(Integer.valueOf(2000), pubSubConfig.getTimeout4Llm(1000));
        Assert.assertEquals("NOTIFIER", pubSubConfig.getNotifier("NOTIFIER2"));
        Assert.assertEquals("FORMAT", pubSubConfig.getFormatter());
        Assert.assertEquals("NOTIFIER", pubSubConfig.getNotifier());
        Assert.assertEquals("REPLAY", pubSubConfig.getReply());
        Assert.assertEquals(Integer.valueOf(2000), pubSubConfig.getTimeout4Llm());
    }

    @Test
    public void testInit() {
        PubSubConfig pubSubConfig = new PubSubConfig();
        Assert.assertNull(pubSubConfig.getNotifier());
        pubSubConfig.init("NOTIFIER");
        Assert.assertEquals("NOTIFIER", pubSubConfig.getNotifier());
    }

    @Test
    public void testMerge() throws Exception {
        PubSubConfig base = new PubSubConfig();
        PubSubConfig other = new PubSubConfig();
        PubSubConfig merged = base.merge(null);
        Assert.assertSame(base, merged);
        base.setTimeout4Llm(100);
        other.setTimeout4Llm(200);
        base.setFormatter("BASE_FORMAT");
        other.setFormatter("OTHER_FORMAT");
        base.setNotifier("BASE_NOTIFIER");
        other.setNotifier("OTHER_NOTIFIER");
        base.setReply("BASE_REPLY");
        other.setReply("OTHER_REPLY");
        base.setCode(201);
        other.setCode(202);
        base.merge(other);
        Assert.assertEquals(Integer.valueOf(100), base.getTimeout4Llm());
        Assert.assertEquals("BASE_FORMAT", base.getFormatter());
        Assert.assertEquals("BASE_NOTIFIER", base.getNotifier());
        Assert.assertEquals("BASE_REPLY", base.getReply());
        Assert.assertEquals(Integer.valueOf(201), base.getCode());
        PubSubConfig base2 = new PubSubConfig();
        PubSubConfig other2 = new PubSubConfig();
        other2.setTimeout4Llm(300);
        other2.setFormatter("OTHER_FORMAT2");
        other2.setNotifier("OTHER_NOTIFIER2");
        other2.setReply("OTHER_REPLY2");
        other2.setCode(203);
        base2.merge(other2);
        Assert.assertEquals(Integer.valueOf(300), base2.getTimeout4Llm());
        Assert.assertEquals("OTHER_FORMAT2", base2.getFormatter());
        Assert.assertEquals("OTHER_NOTIFIER2", base2.getNotifier());
        Assert.assertEquals("OTHER_REPLY2", base2.getReply());
        Assert.assertEquals(Integer.valueOf(203), base2.getCode());
    }

    @Test
    public void testHasFormatterAndReply() {
        PubSubConfig config = new PubSubConfig();
        Assert.assertTrue(config.hasFormatter());
        config.setFormatter("FORMAT");
        Assert.assertFalse(config.hasFormatter());
        Assert.assertFalse(config.hasReply());
        config.setReply("REPLY");
        Assert.assertTrue(config.hasReply());
    }

    @Test
    public void testGetCode() {
        PubSubConfig config = new PubSubConfig();
        Assert.assertEquals(Integer.valueOf(ProtocolCode.C200), config.getCode());
        config.setCode(500);
        Assert.assertEquals(Integer.valueOf(500), config.getCode());
    }

    @Test
    public void testMergeNull() throws Exception {
        PubSubConfig config = new PubSubConfig();
        config.setFormatter("F");
        Assert.assertEquals("F", config.merge(null).getFormatter());
    }
}
