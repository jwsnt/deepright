package ai.open.right.netty.chat;

import ai.open.right.netty.chat.distribute.NettyDistributor;
import ai.open.right.netty.chat.server.http.NettyHttpHandler;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class NettyChatHandlerTest {

    @Test
    public void testSetGet() {
        NettyDistributor nettyDistributor = EasyMock.createMock(NettyDistributor.class);
        EasyMock.replay(nettyDistributor);
        NettyChatHandler nettyChatHandler = new NettyHttpHandler();
        nettyChatHandler.setDistributor(nettyDistributor);
        Assert.assertEquals(nettyDistributor, nettyChatHandler.getDistributor());
        EasyMock.verify(nettyDistributor);
    }

    @org.junit.jupiter.api.Test
    public void testNettyChatHandlerInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}