package ai.open.right.workflow.flow.llm.rag.future;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import lombok.extern.slf4j.Slf4j;

import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

@Slf4j
public class RagAsync extends RagAtOnce {

    protected final Future<Void> future;

    protected final Integer timeout;

    public RagAsync(RagConfig ragConfig, Future<Void> future, Integer timeout) {
        super(ragConfig);
        this.timeout = timeout;
        this.future = future;
    }

    @Override
    public void run() throws Exception {
        try {
            this.future.get(this.timeout, TimeUnit.MILLISECONDS);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            this.failed(e);
        }
        super.run();
    }
}
