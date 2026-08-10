package ai.open.right.workflow.flow.track.impl;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.RedisTemplate;

public class RedisTrackFunCallServiceTest {

    @Test
    public void testInit() throws Exception {
        RedisTemplate<String, Object> template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.replay(template);
        RedisTrackFunCallService.InitConfig service = new RedisTrackFunCallService.InitConfig();
        service.setRedis4funCall(template);
        service.setExpire(1000);
        service.setVersion6_2_0(false);
        RedisTrackFunCallService empty = (RedisTrackFunCallService) service.trackFunCallService();
        Assert.assertEquals(template, empty.getRedis4funCall());
        Assert.assertEquals(false, empty.getVersion6_2_0());
        Assert.assertEquals(Integer.valueOf(1000), empty.getExpire());
        EasyMock.verify(template);
    }
}
