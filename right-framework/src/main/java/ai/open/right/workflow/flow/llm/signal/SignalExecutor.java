package ai.open.right.workflow.flow.llm.signal;

public interface SignalExecutor {

    public static final String SIGNAL_KEY = "signal";

    public String getAndDelContentBuffer(int s, int e);

    public Integer indexOfContentBuffer(String str, int s);

    public Integer indexOfContentBuffer(String str);

    public void setSignalMetadata(String signal);

    public void setWorkflow(String workflow);

    public void setNotifier(String notifier);

    public void silent(Boolean silent);

    public void notify(Boolean notify);
}
