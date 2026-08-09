package ai.deepright.auth;

import ai.open.right.workflow.flow.WorkflowTask;

public interface AuthService {

    public static final String NAME = "auth";

    public void auth(WorkflowTask workTask, String provider, String token) throws Exception;

    public Boolean support(String provider) throws Exception;
}
