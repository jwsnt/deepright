package ai.open.right.workflow.flow.llm.signal;

import ai.open.right.workflow.flow.llm.Message;

import java.util.List;

public interface SignalDistributor {

    public void distribute(SignalConfig signalConfig, String signal, Message message) throws Exception;

    public void distribute(SignalConfig signalConfig, List<String> signal, Message message) throws Exception;
}
