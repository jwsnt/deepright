package ai.deepright.feature;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.router.RouterAgent;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.FilenameUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.Map;
import java.util.regex.Pattern;

public class FeatureUtils {

    public static final Pattern WINDOWS_URI_PATH = Pattern.compile("^/[A-Za-z]:[\\\\/].*");

    public static final String KEY_AGENTS = "agents";

    public static String escapePath(WorkflowTask workTask, String path) throws Exception {
        return FeatureUtils.escapePath(FeatureFlag.isWindows(workTask), path);
    }

    // 仅处理路径中的特殊字符（转义用escapeShell）
    public static String escapePath(Boolean isWindows, String path) throws Exception {
        return StringUtils.isEmpty(path) ? path : (isWindows ? path.replace("\"", "\"\"") : path);
    }

    public static String escapeShell(WorkflowTask workTask, String shell) throws Exception {
        return FeatureUtils.escapeShell(FeatureFlag.isWindows(workTask), shell);
    }

    // 将整个字符串包装成一个安全的Shell参数
    public static String escapeShell(Boolean isWindows, String shell) throws Exception {
        return StringUtils.isEmpty(shell) ? shell : (isWindows ? "\"" + shell.replace("\"", "\"\"") + "\"" : "'" + shell.replace("'", "'\\''") + "'");
    }

    public static String escapeFile(WorkflowTask workTask, String file) throws Exception {
        if (StringUtils.startsWithIgnoreCase(file, "file:///")) {
            String escape = file.substring(7);
            if (FeatureFlag.isWindows(workTask)) {
                // file:///C:/Users/a.txt -> C:/Users/a.txt
                return FeatureUtils.WINDOWS_URI_PATH.matcher(escape).matches() ? escape.substring(1) : escape;
            } else {
                // file:///tmp/a.txt -> /tmp/a.txt
                return StringUtils.startsWith(escape, "/") ? escape : "/" + escape;
            }
        } else {
            return file;
        }
    }

    public static String buildLineSeparator(WorkflowTask workTask) throws Exception {
        return FeatureUtils.buildLineSeparator(FeatureUtils.buildSys(workTask));
    }

    public static String buildLineSeparator(String sys) throws Exception {
        // 有先后顺序，否则win会匹配darwin
        if (sys.contains("mac")) {
            return (sys.contains("os x") || sys.contains("darwin")) ? "\n" : "\r";
        } else if (sys.contains("win")) {
            return "\r\n";
        } else {
            return "\n";
        }
    }

    public static String buildFileSeparator(WorkflowTask workTask) throws Exception {
        return FeatureUtils.buildFileSeparator(FeatureUtils.buildSys(workTask));
    }

    public static String buildFileSeparator(String sys) throws Exception {
        return FeatureFlag.isWindows(sys) ? "\\" : "/";
    }

    public static RouterAgent[] buildRouterAgents(WorkflowTask workTask) throws Exception {
        return JsonUtils.transfer(MapUtils.getObject(workTask.getMetadata(), FeatureUtils.KEY_AGENTS), RouterAgent[].class);
    }

    // 目标服务商
    public static String buildTargetProvider(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), ProviderRequestService.KEY_PROVIDER);
    }

    // 原始服务商
    public static String buildSourceProvider(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_PROVIDER, FeatureUtils.buildTargetProvider(workTask));
    }

    public static String buildSandBoxPath(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_SANDBOX_PATH);
    }

    public static Long buildLastResponse(WorkflowTask workTask) throws Exception {
        return MapUtils.getLong(workTask.getMetadata(), FeatureField.KEY_LAST_RESPONSE);
    }

    public static String buildWorkspace(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_WORKSPACE);
    }

    public static String buildKnowledge(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_KNOWLEDGE_CONTENT);
    }

    public static String buildTerminal(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_TERMINAL);
    }

    public static String buildGateway(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_GATEWAY);
    }

    public static String buildTimezone(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_TIMEZONE), "");
    }

    public static String buildAgentId(WorkflowTask workTask) throws Exception {
        return MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_AGENTID);
    }

    public static String buildUser(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_USER), "");
    }

    public static String buildSoul(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_SOUL), "");
    }

    public static String buildApp(Map<String, Object> metadata) throws Exception {
        return MapUtils.getString(metadata, FeatureField.KEY_APP);
    }

    public static String buildApp(WorkflowTask workTask) throws Exception {
        return FeatureUtils.buildApp(workTask.getMetadata());
    }

    public static String buildSys(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_SYS), "darwin");
    }

    // 系统路径
    public static String buildSysPath(WorkflowTask workTask, String path) throws Exception {
        Boolean isWindows = FeatureFlag.isWindows(workTask);
        return FeatureUtils.escapePath(isWindows, isWindows ? FilenameUtils.separatorsToWindows(path) : FilenameUtils.separatorsToUnix(path));
    }
}
