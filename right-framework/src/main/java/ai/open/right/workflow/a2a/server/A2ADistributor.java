package ai.open.right.workflow.a2a.server;

import ai.open.right.workflow.a2a.A2ARequest;

// 分发A2A
public interface A2ADistributor {

    public void distribute(A2ARequest a2aRequest) throws Exception;
}
