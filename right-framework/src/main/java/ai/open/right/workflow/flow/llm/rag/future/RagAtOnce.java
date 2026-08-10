package ai.open.right.workflow.flow.llm.rag.future;

import ai.open.right.workflow.flow.llm.rag.RagConfig;
import lombok.Getter;
import lombok.extern.slf4j.Slf4j;

@Slf4j
public class RagAtOnce implements RagFuture {

    @Getter
    protected final RagConfig ragConfig;

    @Getter
    protected Boolean success = true;

    protected Exception exception;

    public RagAtOnce(RagConfig ragConfig) {
        this.ragConfig = ragConfig;
    }

    @Override
    public void run() throws Exception {
        if (!this.success) {
            throw this.exception;
        }
    }

    @Override
    public RagConfig config() {
        return this.ragConfig;
    }

    public void failed(Exception exception) {
        this.exception = exception;
        this.success = false;
    }
}
