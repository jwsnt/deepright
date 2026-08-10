package ai.open.right.workflow.flow.track;

import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
public class TrackFunCall {

    protected TrackDimension trackDimension;

    protected Object response;

    protected Object request;
}
