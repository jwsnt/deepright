package ai.open.right.workflow.flow.script;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.config.TimeoutConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class ScriptConfig extends GlobalConfig {

    public static final String ENGINE_JAVASCRIPT = "javascript";

    public static final String ENGINE_POLYGLOT = "polyglot";

    public static final String ENGINE_COMMAND = "command";

    public static final String ENGINE_PYTHON = "python";

    public static final String ENGINE_JYTHON = "jython";

    public static final String ENGINE_LUA = "lua";

    public static final String WRAP_OBJECT = "object";

    // 脚本校准
    protected ScriptCorrector corrector;

    // 执行超时
    protected TimeoutConfig timeout;

    // 默认Success Code
    protected Integer successCode;

    // 用于条件判断的思考链（Workflow）
    protected String condition;

    // 通知方式（Localhost/Endpoint/Source）
    protected String notifier;

    // 脚本解析器
    protected String engine;

    // 是否包装脚本响应
    protected String wrap;

    public ScriptConfig merge(ScriptConfig scriptConfig) throws Exception {
        super.merge(scriptConfig);
        if (scriptConfig != null) {
            this.successCode = this.successCode != null ? this.successCode : scriptConfig.successCode;
            this.condition = StringUtils.defaultIfBlank(this.condition, scriptConfig.condition);
            this.corrector = this.corrector != null ? this.corrector : scriptConfig.corrector;
            this.notifier = StringUtils.defaultIfBlank(this.notifier, scriptConfig.notifier);
            this.engine = StringUtils.defaultIfBlank(this.engine, scriptConfig.engine);
            this.timeout = this.timeout != null ? this.timeout : scriptConfig.timeout;
            this.wrap = StringUtils.defaultIfBlank(this.wrap, scriptConfig.wrap);
        }
        return this;
    }

    public ScriptConfig init(String notifier) {
        this.notifier = StringUtils.defaultString(this.notifier, notifier);
        return this;
    }

    public Boolean hasCorrector() {
        return this.corrector != null;
    }

    public Boolean hasCondition() {
        return !StringUtils.isEmpty(this.condition);
    }

    public Integer getTimeout4Condition(Integer timeout4Condition) {
        if (this.timeout == null) {
            return timeout4Condition;
        }
        return this.timeout.getTimeout4Condition(timeout4Condition);
    }

    public Integer getTimeout4Corrector(Integer timeout4Corrector) {
        if (this.timeout == null) {
            return timeout4Corrector;
        }
        return this.timeout.getTimeout4Corrector(timeout4Corrector);
    }

    public Integer getTimeout(Integer timeout) {
        if (this.timeout == null) {
            return timeout;
        }
        return this.timeout.getTimeout(timeout);
    }

    public Integer getSuccessCode() {
        return this.successCode != null ? this.successCode : ProtocolCode.C200;
    }

    public String getEngine() {
        return this.engine != null ? this.engine : ScriptConfig.ENGINE_PYTHON;
    }

    public Boolean shouldWrap() {
        return !StringUtils.isEmpty(this.wrap);
    }

    public Boolean hasNotifier() {
        return !StringUtils.isEmpty(this.notifier);
    }

    public Boolean isSuccessCode(Integer code) {
        return this.getSuccessCode().equals(code);
    }
}
