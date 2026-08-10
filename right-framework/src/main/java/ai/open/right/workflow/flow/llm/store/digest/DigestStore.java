package ai.open.right.workflow.flow.llm.store.digest;

import ai.open.right.workflow.flow.llm.store.Dimension;

public interface DigestStore {

    public Digest upsert(Dimension dimension, String scene, Digest digest);
}

