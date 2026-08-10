package ai.open.right.workflow.a2a.server;

import ai.open.right.workflow.a2a.A2ARequest;

public interface A2ACmdExportService {

    // 是否可以处理该请求
    public Boolean support(A2ARequest a2aRequest) throws Exception;

    public void cmd(A2ARequest a2aRequest) throws Exception;
}
