package ai.open.right.workflow.flow.track;

import org.junit.Assert;
import org.junit.Test;

public class TrackFunCallTest {

    @Test
    public void test() {
        TrackDimension trackDimension = new TrackDimension();
        TrackFunCall trackFunCall = new TrackFunCall();
        trackFunCall.setTrackDimension(trackDimension);
        trackFunCall.setResponse("REP");
        trackFunCall.setRequest("REQ");
        Assert.assertEquals(trackDimension, trackFunCall.getTrackDimension());
        Assert.assertEquals("REP", trackFunCall.getResponse());
        Assert.assertEquals("REQ", trackFunCall.getRequest());
    }
}
