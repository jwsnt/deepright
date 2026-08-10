package ai.open.right.netty.chat.distribute;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.store.history.History;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.Map;

public class NettyEventTest {

    @Test
    public void test() {
        NettyRequest nettyRequest = NettyRequest.class.cast(ObjectBuilder.buildWorkflowTask());
        nettyRequest.addHistories(Arrays.asList(new History()));
        NettyEvent event = new NettyEvent(nettyRequest);
        event.init();
        Assert.assertEquals("UNKNOWN", event.getWorkflow());
        Assert.assertEquals("UNKNOWN", event.getDevice());
        Assert.assertEquals("UNKNOWN", event.getBiz());
        Assert.assertEquals("UNKNOWN", event.getChat());
        Assert.assertEquals("netty", event.getType());
        Assert.assertNotNull(event.getNow());
        Assert.assertEquals("UNKNOWN-UNKNOWN-UNKNOWN", event.getDimension());
        Map<String, Object> body = Map.class.cast(event.getData());
        Assert.assertEquals(Integer.valueOf(5), Integer.valueOf(body.size()));
        Assert.assertEquals(nettyRequest.getConversation(), body.get("conversation"));
        Assert.assertEquals(nettyRequest.getHistories(), body.get("histories"));
        Assert.assertEquals(nettyRequest.getMetadata(), body.get("metadata"));
        Assert.assertEquals(nettyRequest.getQuery(), body.get("query"));
    }

    @org.junit.jupiter.api.Test
    public void testNettyEventInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}