package ai.deepright.router;

import ai.open.right.workflow.flow.WorkflowTask;

import java.util.List;

public interface RouterService {

    // 设备是否有心跳
    public Boolean hasHeartbeat(WorkflowTask workTask, String agent) throws Exception;

    public Boolean hasHeartbeat(RouterDevice routerDevice) throws Exception;

    public Boolean hasHeartbeat(WorkflowTask workTask) throws Exception;

    // 更新心跳，异步
    public void heartbeat(WorkflowTask workTask);

    public RouterDevice fetch(WorkflowTask workTask, String device, String agent) throws Exception;

    public RouterDevice fetch(WorkflowTask workTask, String agent) throws Exception;

    public RouterDevice fetch(String router, String agent) throws Exception;

    public RouterDevice fetch(RouterDevice routerDevice) throws Exception;

    public RouterDevice fetch(WorkflowTask workTask) throws Exception;

    public RouterDevice fetch(String key) throws Exception;

    // 获取设备集
    public List<RouterDevice> router(WorkflowTask workTask) throws Exception;
}
