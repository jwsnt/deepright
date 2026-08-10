package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.workflow.flow.llm.store.Dimension;

import java.util.List;

public interface HistoryStore {

    public void store(Dimension dimension, List<String> repositories, String query, String answer, String reasoning, Integer expire, Integer nums, Long now) throws Exception;

    public void store(Dimension dimension, List<String> repositories, String query, String answer, Integer expire, Integer nums, Long now) throws Exception;

    public void store(Dimension dimension, List<String> repositories, List<HistoryPair> pairs, Integer expire, Integer nums) throws Exception;

    public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception;

    public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now, Long offset) throws Exception;

    // 当前时间前/后的N条，desc=true, now=当前时间的负数，用于获取指定时间后的N条
    // 如果指定now则为开区间(now,0]
    public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now) throws Exception;

    public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc) throws Exception;

    // 当前时间前的N条
    public List<History> restore(Dimension dimension, String scene, Integer nums, Long now) throws Exception;

    public List<History> restore(Dimension dimension, String scene, Integer nums) throws Exception;

    public void clear(Dimension dimension, List<String> repositories, Boolean desc, Long now) throws Exception;

    public void clear(Dimension dimension, List<String> repositories, Long now) throws Exception;
}
