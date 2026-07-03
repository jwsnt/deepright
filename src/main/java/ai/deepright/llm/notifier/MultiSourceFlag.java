package ai.deepright.llm.notifier;

// 客户端标记
public interface MultiSourceFlag {

    public static final String KEY_START = "__TASK_START__";

    public static final String KEY_CLOSE = "__TASK_CLOSE__";

    public static final String PROCESS = "__PROCESS__";

    public static final String TARGET = "__TARGET__";

    public static final String DELAY = "__DELAY__";

    public static final String RESET = "__RESET__";

    public static final String WARN = "__WARN__";
}
