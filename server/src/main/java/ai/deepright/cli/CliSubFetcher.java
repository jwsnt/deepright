package ai.deepright.cli;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.deepright.router.RouterDevice;

import java.util.List;

public interface CliSubFetcher {

    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, Boolean waitPub, String suffix, String device, String agent, String cmd, String why) throws Exception;

    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, Boolean waitPub, String suffix, String device, String cmd, String why) throws Exception;

    public CliPubData command(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, Boolean waitPub, String cmd, String why) throws Exception;

    public CliPubData command(WorkflowTask workTask, RouterDevice router, Boolean waitPub, String suffix, String cmd, String why) throws Exception;

    // CLI下发，Cmd 执行命令
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, Boolean waitPub, String device, String cmd, String why) throws Exception;

    public CliPubData command(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, String cmd, String why) throws Exception;

    // CLI下发，Cmd 执行命令
    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, String device, String cmd, String why) throws Exception;

    public CliPubData command(WorkflowTask workTask, CliSubOps subOps, String cmd, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String suffix, String device, String agent, String path, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String suffix, String device, String agent, List<String> paths, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String suffix, String device, String path, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, String path, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, CliSubOps subOps, List<String> paths, String why) throws Exception;

    // CLI下发，Fetch 获取文件
    public CliPubData fetch(WorkflowTask workTask, CliSubOps subOps, String device, String path, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, String path, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, RouterDevice router, java.util.List<String> paths, String why) throws Exception;

    // CLI下发，Fetch 获取文件
    public CliPubData fetch(WorkflowTask workTask, String device, String path, String why) throws Exception;

    public CliPubData fetch(WorkflowTask workTask, String path, String why) throws Exception;

    // CliGet/CliSub
    public static String getDeviceKey(String key) throws Exception {
        return CliSubFetcher.class.getSimpleName() + key + "r";
    }

    public static String getTidKey(String key) throws Exception {
        return CliSubFetcher.class.getSimpleName() + key + "t";
    }
}
