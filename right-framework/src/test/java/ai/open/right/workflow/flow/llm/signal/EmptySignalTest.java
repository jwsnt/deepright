package ai.open.right.workflow.flow.llm.signal;

import org.junit.Test;

public class EmptySignalTest {

    @Test
    public void test() throws Exception {
        SignalStream.EmptySignal emptySignal = new SignalStream.EmptySignal();
        emptySignal.finish(null);
        emptySignal.signal(null, null);
    }
}
