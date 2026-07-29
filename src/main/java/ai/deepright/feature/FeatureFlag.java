package ai.deepright.feature;

import ai.open.right.workflow.flow.WorkflowTask;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;

public class FeatureFlag {

    public static final Pattern WINDOWS_DRIVE_PATH = Pattern.compile("^[A-Za-z]:[\\\\/].*");

    // 是否为主动更新知识库请求
    public static final String KEY_KNOWLEDGE_COMMIT = "knowledge_commit";

    // 是否为主动更新偏好请求
    public static final String KEY_PROFILE_COMMIT = "profile_commit";

    // 是否为技能提取
    public static final String KEY_SKILL_EXTRACT = "skill_extract";

    public static final String KEY_PLUGINS = "plugins";

    public static final String KEY_VERIFY = "verify";

    // 是否开启了HTML输出
    public static final String KEY_HTML = "html";

    public static final String KEY_TEST = "test";

    // 是否激活指定插件
    public static Boolean isActivePlugin(WorkflowTask workTask, String plugin) throws Exception {
        List<String> plugins = List.class.cast(MapUtils.getObject(workTask.getMetadata(), FeatureFlag.KEY_PLUGINS));
        return !CollectionUtils.isEmpty(plugins) && plugins.contains(plugin);
    }

    // 是否存在自定义Response Schema
    public static Boolean isResponseSchema(Map<String, Object> metadata) throws Exception {
        return !StringUtils.isEmpty(MapUtils.getString(metadata, FeatureField.KEY_RESPONSE_SCHEMA));
    }

    // 是否为更新知识库
    public static Boolean isKnowledgeCommit(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureFlag.KEY_KNOWLEDGE_COMMIT, false);
    }

    public static Boolean isProfileCommit(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureFlag.KEY_PROFILE_COMMIT, false);
    }

    public static Boolean isResponseSchema(WorkflowTask workTask) throws Exception {
        return FeatureFlag.isResponseSchema(workTask.getMetadata());
    }

    public static Boolean isSkillExtract(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureFlag.KEY_SKILL_EXTRACT, false);
    }

    public static Boolean isOpenTeamMode(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureField.KEY_ROUTER_STARTUP, false);
    }

    public static Boolean isAbsolutePath(WorkflowTask workTask, String path) throws Exception {
        return FeatureFlag.isAbsolutePath(FeatureFlag.isWindows(workTask), path);
    }

    public static Boolean isAbsolutePath(boolean isWindows, String path) {
        if (!StringUtils.isEmpty(path)) {
            if (isWindows) {
                // C:\foo / C:/foo / \\server\share\foo
                return FeatureFlag.WINDOWS_DRIVE_PATH.matcher(path).matches() || path.startsWith("\\\\");
            } else {
                return path.startsWith("/");
            }
        } else {
            return false;
        }
    }

    public static Boolean isSandbox(WorkflowTask workTask) throws Exception {
        return !StringUtils.isEmpty(FeatureUtils.buildSandBoxPath(workTask));
    }

    // 与cli.go/get请求一致，{"sys": runtime.GOOS}，据此判断是否为Windows
    public static Boolean isWindows(WorkflowTask workTask) throws Exception {
        return FeatureFlag.isWindows(StringUtils.trim(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_SYS)));
    }

    public static Boolean isWindows(String sys) throws Exception {
        return StringUtils.containsIgnoreCase(sys, "windows");
    }

    public static Boolean isVerify(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureFlag.KEY_VERIFY, false);
    }

    public static Boolean isMacOs(WorkflowTask workTask) throws Exception {
        return FeatureFlag.isMacOs(FeatureUtils.buildSys(workTask));
    }

    public static Boolean isMacOs(String sys) throws Exception {
        return StringUtils.containsIgnoreCase(sys, "darwin");
    }

    public static Boolean isLogic(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureField.KEY_LOGIC, false);
    }

    public static Boolean isWsl(WorkflowTask workTask) throws Exception {
        return FeatureFlag.isWsl(FeatureUtils.buildSys(workTask));
    }

    public static Boolean isWsl(String sys) throws Exception {
        return StringUtils.containsIgnoreCase(sys, FeatureField.KEY_WSL);
    }

    // 是否为后台线程
    public static Boolean isDaemon(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureField.KEY_DAEMON, false);
    }

    // 是否需要静默（阻止回源消息）
    public static Boolean isSilent(Map<String, Object> metadata) throws Exception {
        // 主动开启静默，或标记使用自定义Response Schema
        return MapUtils.getBoolean(metadata, FeatureField.KEY_SILENT, false) || FeatureFlag.isResponseSchema(metadata);
    }

    public static Boolean isSilent(WorkflowTask workTask) throws Exception {
        // 主动开启静默，或标记使用自定义Response Schema
        return FeatureFlag.isSilent(workTask.getMetadata()) || FeatureFlag.isResponseSchema(workTask);
    }

    public static Boolean isHtml(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureFlag.KEY_HTML, false);
    }

    public static Boolean isTask(WorkflowTask workTask) throws Exception {
        return MapUtils.getObject(workTask.getMetadata(), FeatureField.KEY_TASK) != null;
    }

    // 是否来自周期任务
    public static Boolean isCron(WorkflowTask workTask) throws Exception {
        String cron = MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_CRON_TYPE);
        // 存在cron_type标记，如果来自插件名则为插件名，如feishu
        return !StringUtils.isEmpty(cron);
    }

    // 是否是测试请求
    public static Boolean isTest(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureFlag.KEY_TEST, false);
    }
}
