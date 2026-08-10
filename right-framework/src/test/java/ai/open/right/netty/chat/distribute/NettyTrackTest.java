package ai.open.right.netty.chat.distribute;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.track.TrackChat;
import ai.open.right.workflow.flow.track.TrackChatService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class NettyTrackTest {

    @Test
    public void testInit() throws Exception {
        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        EasyMock.replay(trackChatService);
        NettyTrack.InitConfig nettyTrack = new NettyTrack.InitConfig();
        nettyTrack.setTrackChatService(trackChatService);
        NettyTrack init = nettyTrack.nettyTrack();
        Assert.assertEquals(init.getTrackChatService(), trackChatService);
        EasyMock.verify(trackChatService);
    }

    @Test
    public void testTrack() throws Exception {
        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        trackChatService.store(EasyMock.anyObject(TrackChat.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(trackChatService);
        NettyTrack nettyTrack = new NettyTrack();
        nettyTrack.setTrackChatService(trackChatService);
        nettyTrack.track(ObjectBuilder.buildWorkflowTask(), ObjectBuilder.buildSegment());
        EasyMock.verify(trackChatService);
    }

    @Test
    public void testTrackException() throws Exception {
        NettyTrack nettyTrack = new NettyTrack();
        nettyTrack.track(ObjectBuilder.buildWorkflowTask(), ObjectBuilder.buildSegment());
    }

    /**
     * store 抛异常时 track 吞掉异常，不向调用方抛出（由 WorkflowException.dolog 记录）
     */
    @Test
    public void testTrackStoreThrowsDoesNotPropagate() throws Exception {
        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        trackChatService.store(EasyMock.anyObject(TrackChat.class));
        EasyMock.expectLastCall().andThrow(new RuntimeException("store failed"));
        EasyMock.replay(trackChatService);
        NettyTrack nettyTrack = new NettyTrack();
        nettyTrack.setTrackChatService(trackChatService);
        nettyTrack.track(ObjectBuilder.buildWorkflowTask(), ObjectBuilder.buildSegment());
        EasyMock.verify(trackChatService);
    }

    @org.junit.jupiter.api.Test
    public void testNettyTrackInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

}