package ai.open.right.workflow.flow.track;

import java.util.List;

public interface TrackFunCallService {

    public List<TrackFunCall> restore(TrackDimension trackDimension) throws Exception;

    public void store(TrackFunCall trackFunCall) throws Exception;
}
