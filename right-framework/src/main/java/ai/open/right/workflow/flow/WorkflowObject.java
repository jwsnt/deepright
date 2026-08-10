package ai.open.right.workflow.flow;

public interface WorkflowObject {

    // 指定对象转为思考链（Workflow）当前Query
    public void setObjectQuery(Object object) throws Exception;

    // 思考链（Workflow）当前Query转为指定对象
    public <T> T getObjectQuery(Class<T> clazz) throws Exception;
}
