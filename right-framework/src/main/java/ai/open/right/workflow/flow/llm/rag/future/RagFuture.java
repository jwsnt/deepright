package ai.open.right.workflow.flow.llm.rag.future;

import ai.open.right.workflow.flow.llm.rag.RagConfig;
import lombok.extern.slf4j.Slf4j;

public interface RagFuture {

    public static final RagFuture NOTHING = new RagNothing();

    public void run() throws Exception;

    public RagConfig config();

    @Slf4j
    public class RagNothing implements RagFuture {

        @Override
        public void run() throws Exception {
            if (log.isInfoEnabled()) {
                log.info("Rag nothing");
            }
        }

        @Override
        public RagConfig config() {
            return null;
        }
    }
}
