package ai.open.right.netty.chat.distribute;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.flow.track.TrackChatService;

public class NettyTrackInitConfigTest {

    @Test
    public void shouldCreateNettyTrackWithInjectedService() throws Exception {
        NettyTrack.InitConfig init = new NettyTrack.InitConfig();

        TrackChatService trackChatService = EasyMock.createMock(TrackChatService.class);
        init.setTrackChatService(trackChatService);

        NettyTrack bean = init.nettyTrack();
        Assert.assertNotNull(bean);
        Assert.assertSame(trackChatService, bean.getTrackChatService());
    }
}
