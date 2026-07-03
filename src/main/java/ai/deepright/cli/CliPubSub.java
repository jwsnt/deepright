package ai.deepright.cli;

import ai.open.right.context.UserContext;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.GzipUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.FileStore;
import ai.deepright.feature.FeatureUtils;
import org.apache.commons.lang3.RandomStringUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.nio.file.Paths;

public interface CliPubSub {

    public static final String FUN = "fun";

    public static final String CMD = "cmd";

    // 检查Device
    public static void checkValid(WorkflowTask workTask) throws Exception {
        // 设备号不能为Null或默认值
        Assert.isTrue(!StringUtils.equalsIgnoreCase(workTask.getDevice(), UserContext.UNKNOWN) && !StringUtils.isEmpty(workTask.getDevice()), "The cli device can not be empty");
    }

    // 构建推送CMD
    public static String buildPushCmd(WorkflowTask workTask, FileStore fileStore, Boolean binary, Integer oversize, String content, String file) throws Exception {
        StringBuffer cmd = new StringBuffer();
        cmd.append("mkdir -p ").append(FeatureUtils.escapeShell(workTask, FeatureUtils.buildSysPath(workTask, Path.of(file).getParent().toString()))).append(" && ");
        byte[] data = content.getBytes(StandardCharsets.UTF_8);
        if (binary || BytesUtils.utf8Bytes(content) > oversize) {
            // 二进制文件下发（先压缩）
            cmd.append("curl ").append(FeatureUtils.escapeShell(workTask, fileStore.store(GzipUtils.compress(data), file, workTask))).append(" | gunzip > ").append(FeatureUtils.escapeShell(workTask, file));
        } else {
            String mark = "EOF_" + RandomStringUtils.randomAlphabetic(3);
            cmd.append("cat <<'").append(mark).append("' > ").append(FeatureUtils.escapeShell(workTask, file)).append(System.lineSeparator()).append(content).append(System.lineSeparator()).append(mark);
        }
        return cmd.toString();
    }

    // 构建推送CMD
    public static String buildPushCmd(WorkflowTask workTask, FileStore fileStore, Integer oversize, String content, String file) throws Exception {
        return CliPubSub.buildPushCmd(workTask, fileStore, false, oversize, content, file);
    }

    // 构建推送URL
    public static String buildPushURL(WorkflowTask workTask, String url, String file) throws Exception {
        StringBuffer cmd = new StringBuffer();
        cmd.append("mkdir -p ").append(FeatureUtils.escapeShell(workTask, FeatureUtils.buildSysPath(workTask, Paths.get(file).getParent().toString()))).append(" && ");
        cmd.append("curl ").append(FeatureUtils.escapeShell(workTask, url)).append(" > ").append(FeatureUtils.escapeShell(workTask, file));
        return cmd.toString();
    }
}
