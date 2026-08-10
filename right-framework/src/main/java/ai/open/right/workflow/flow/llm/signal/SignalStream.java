package ai.open.right.workflow.flow.llm.signal;

import ai.open.right.workflow.flow.llm.Message;

public interface SignalStream {

    public static final SignalStream EMPTY = new EmptySignal();

    public void signal(SignalExecutor signalExecutor, Message message) throws Exception;

    public void finish(Message message) throws Exception;

    public static class EmptySignal implements SignalStream {

        @Override
        public void signal(SignalExecutor signalExecutor, Message message) throws Exception {

        }

        public void finish(Message message) throws Exception {

        }
    }
}
