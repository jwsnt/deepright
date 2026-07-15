package ai.deepright.feature;

public interface FeatureField {

    public static final String KEY_KNOWLEDGE_CONTENT = "knowledge_content";

    public static final String KEY_KNOWLEDGE_COMMIT = "knowledge_commit";

    // 自定义Response Schema
    public static final String KEY_RESPONSE_SCHEMA = "response_schema";

    public static final String KEY_ROUTER_UPSTREAM = "router_upstream";

    // 是否已经启动团队
    public static final String KEY_ROUTER_STARTUP = "router_startup";

    // 是否主动关闭团队
    public static final String KEY_ROUTER_DISABLE = "router_disable";

    public static final String KEY_ROUTER_DESC = "router_desc";

    public static final String KEY_LAST_RESPONSE = "lastResponse";

    public static final String KEY_SANDBOX_PATH = "sandbox_path";

    public static final String KEY_FILE_SYSTEM = "file_system";

    public static final String KEY_PLUGINS_DIR = "plugins_dir";

    public static final String KEY_CRON_TYPE = "cron_type";

    public static final String KEY_KNOWLEDGE = "knowledge";

    public static final String KEY_WORKSPACE = "workspace";

    public static final String KEY_TERMINAL = "terminal";

    public static final String KEY_TIMEZONE = "timezone";

    // 是否开启思考模式
    public static final String KEY_THINKING = "thinking";

    public static final String KEY_PROVIDER= "provider";

    public static final String KEY_GATEWAY = "gateway";

    public static final String KEY_AGENTID = "agentId";

    public static final String KEY_VERIFY = "verify";

    // 后台进程（刷新知识库或Task等）
    public static final String KEY_DAEMON = "daemon";

    public static final String KEY_OUTPUT = "output";

    public static final String KEY_SKILLS = "skills";

    public static final String KEY_SILENT = "silent";

    public static final String KEY_LOGIC = "logic";

    public static final String KEY_MEDIA = "media";

    // 是否开启了TASK模式
    public static final String KEY_TASK = "task";

    public static final String KEY_USER = "user";

    public static final String KEY_SOUL = "soul";

    public static final String KEY_SYS = "sys";

    public static final String KEY_APP = "app";

    public static final String KEY_GIT = "git";
}
