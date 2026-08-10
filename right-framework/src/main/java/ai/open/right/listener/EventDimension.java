package ai.open.right.listener;

import ai.open.right.workflow.flow.llm.store.Dimension;

public interface EventDimension extends Dimension {

    public Long getNow();
}
