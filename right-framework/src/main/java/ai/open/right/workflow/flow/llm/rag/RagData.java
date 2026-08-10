package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

import java.util.concurrent.locks.ReentrantLock;

@Setter
@Getter
@Builder
public class RagData {

    private final ReentrantLock lock = new ReentrantLock();

    protected final ProviderRequest request;

    protected final LLMConfig config;

    protected final LLMQuery query;

    protected String prompt;

    public void unlock() {
        this.lock.unlock();
    }

    public void lock() {
        this.lock.lock();
    }
}
