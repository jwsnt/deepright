package ai.deepright.memory;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;

import java.util.List;

public interface MemoryService {

    public static final String LANG_KEY_MEMORY_INIT_FILE = "memory.init.file";


    public static final String LANG_KEY_MEMORY_RECALL_VALID = "memory.recall.valid";

    public static final String NAME = "fun_memory_recall";

    // 初始化记忆
    public String init(WorkflowTask workTask) throws Exception;

    public String recall(WorkflowTask workTask, MemoryRecall recall) throws Exception;

    public void commit(WorkflowTask workTask) throws Exception;

    public void refresh(WorkflowTask workTask, List<History> histories) throws Exception;

    public Boolean support(WorkflowTask workTask) throws Exception;
}
