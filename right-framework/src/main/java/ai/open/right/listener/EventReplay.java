package ai.open.right.listener;

import ai.open.right.workflow.flow.llm.store.Dimension;

import java.util.List;

public interface EventReplay {

    public List<Event> replay(Dimension dimension) throws Exception;
}
