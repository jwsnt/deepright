package ai.open.right.workflow.flow.file;

import ai.open.right.workflow.flow.WorkflowTask;

public interface FileStore {

    public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception;

    public String store(byte[] bytes, String suffix) throws Exception;

    // 是否支持网络
    public Boolean supportNetwork() throws Exception;

    // 是否支持磁盘
    public Boolean supportFilesys() throws Exception;

    public String name() throws Exception;
}
