package ai.open.right.workflow.flow.track;

import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.DimensionImpl;
import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
public class TrackDimension extends DimensionImpl {

    protected String track;

    public TrackDimension(Dimension dimension, String track) {
        this.setWorkflow(dimension.getWorkflow());
        this.setDevice(dimension.getDevice());
        this.setChat(dimension.getChat());
        this.setBiz(dimension.getBiz());
        this.track = track;
    }

    public TrackDimension() {

    }
}
