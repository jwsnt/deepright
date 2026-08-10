package ai.open.right.workflow.flow;

import lombok.Builder;
import org.apache.commons.lang3.time.StopWatch;

@Builder
public class WorkflowWatcher {

    private final StopWatch stopWatch = StopWatch.createStarted();

    public Long getConsuming() {
        this.stopWatch.split();
        return this.stopWatch.getSplitTime();
    }
}
