package ai.open.right.workflow.flow.track.impl;

import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.track.TrackDimension;
import ai.open.right.workflow.flow.track.TrackFunCall;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.data.redis.core.ListOperations;
import org.springframework.data.redis.core.RedisTemplate;

import java.util.Arrays;
import java.util.List;

public class RedisTrackServiceTest {

    @Test
    public void testKey() throws Exception {
        TrackDimension trackDimension = new TrackDimension();
        trackDimension.setTrack("T");
        trackDimension.setBiz("B");
        trackDimension.setChat("C");
        trackDimension.setDevice("D");
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        Assert.assertEquals("rightRedisTrackFunCallServiceBCDT", redisTrackService.getKey(trackDimension));
    }

    @Test
    public void testStore() throws Exception {
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisTrackSetCallback.class))).andReturn(null).anyTimes();
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        redisTrackService.setRedis4funCall(template);
        EasyMock.replay(template);
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        redisTrackService.store(trackFunCall);
        EasyMock.verify(template);
    }

    @Test
    public void testStoreWithException() throws Exception {
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        redisTrackService.store(trackFunCall);
    }

    @Test
    public void testReStore1() throws Exception {
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ListOperations listOperations = EasyMock.createMock(ListOperations.class);
        EasyMock.expect(listOperations.rightPop(EasyMock.anyObject(String.class), EasyMock.anyInt())).andReturn(Arrays.asList(GzipUtils.compress(JsonUtils.write(trackFunCall)), GzipUtils.compress(JsonUtils.write(trackFunCall)))).anyTimes();
        EasyMock.expect(template.opsForList()).andReturn(listOperations).anyTimes();
        EasyMock.expect(template.executePipelined(EasyMock.anyObject(RedisTrackSetCallback.class))).andReturn(null).anyTimes();
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        redisTrackService.setRedis4funCall(template);
        redisTrackService.setVersion6_2_0(true);
        EasyMock.replay(template, listOperations);
        List<TrackFunCall> result = redisTrackService.restore(trackDimension);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(result.size()));
        Assert.assertEquals("REP", result.getFirst().getResponse());
        Assert.assertEquals("REQ", result.getFirst().getRequest());
        EasyMock.verify(template, listOperations);
    }

    @Test
    public void testReStore2() throws Exception {
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ListOperations listOperations = EasyMock.createMock(ListOperations.class);
        EasyMock.expect(listOperations.rightPop(EasyMock.anyObject(String.class))).andReturn(GzipUtils.compress(JsonUtils.write(trackFunCall))).times(5).andReturn(null).anyTimes();
        EasyMock.expect(template.opsForList()).andReturn(listOperations).anyTimes();
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        redisTrackService.setRedis4funCall(template);
        redisTrackService.setVersion6_2_0(false);
        EasyMock.replay(template, listOperations);
        List<TrackFunCall> result = redisTrackService.restore(trackDimension);
        Assert.assertEquals(Integer.valueOf(5), Integer.valueOf(result.size()));
        Assert.assertEquals("REP", result.getFirst().getResponse());
        Assert.assertEquals("REQ", result.getFirst().getRequest());
        EasyMock.verify(template, listOperations);
    }


    @Test
    public void testReStoreWithEmpty() throws Exception {
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ListOperations listOperations = EasyMock.createMock(ListOperations.class);
        EasyMock.expect(listOperations.rightPop(EasyMock.anyObject(String.class))).andReturn(null).anyTimes();
        EasyMock.expect(template.opsForList()).andReturn(listOperations).anyTimes();
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        redisTrackService.setRedis4funCall(template);
        redisTrackService.setVersion6_2_0(false);
        EasyMock.replay(template, listOperations);
        List<TrackFunCall> result = redisTrackService.restore(trackDimension);
        Assert.assertEquals(Integer.valueOf(0), Integer.valueOf(result.size()));
        EasyMock.verify(template, listOperations);
    }


    @Test
    public void testReStoreWithNotJson() throws Exception {
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ListOperations listOperations = EasyMock.createMock(ListOperations.class);
        EasyMock.expect(listOperations.rightPop(EasyMock.anyObject(String.class))).andReturn(GzipUtils.compress(trackFunCall.toString())).times(5).andReturn(null).anyTimes();
        EasyMock.expect(template.opsForList()).andReturn(listOperations).anyTimes();
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        redisTrackService.setRedis4funCall(template);
        redisTrackService.setVersion6_2_0(false);
        EasyMock.replay(template, listOperations);
        List<TrackFunCall> result = redisTrackService.restore(trackDimension);
        Assert.assertEquals(Integer.valueOf(0), Integer.valueOf(result.size()));
        EasyMock.verify(template, listOperations);
    }

    @Test
    public void testReStoreWithException() throws Exception {
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        RedisTemplate template = EasyMock.createMock(RedisTemplate.class);
        ListOperations listOperations = EasyMock.createMock(ListOperations.class);
        EasyMock.expect(listOperations.rightPop(EasyMock.anyObject(String.class))).andThrow(new RuntimeException()).anyTimes();
        EasyMock.expect(template.opsForList()).andReturn(listOperations).anyTimes();
        RedisTrackFunCallService redisTrackService = new RedisTrackFunCallService();
        redisTrackService.setRedis4funCall(template);
        redisTrackService.setVersion6_2_0(false);
        EasyMock.replay(template, listOperations);
        List<TrackFunCall> result = redisTrackService.restore(trackDimension);
        Assert.assertEquals(Integer.valueOf(0), Integer.valueOf(result.size()));
        EasyMock.verify(template, listOperations);
    }

}
