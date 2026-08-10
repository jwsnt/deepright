package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.config.TimeoutConfig;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.digest.RagDigestConfig;
import ai.open.right.workflow.flow.llm.rag.mcp.RagMcpConfig;
import ai.open.right.workflow.flow.llm.rag.meta.RagMetaConfig;
import ai.open.right.workflow.flow.llm.rag.remote.RagRemoteConfig;
import ai.open.right.workflow.flow.llm.rag.skills.RagSkillsConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.BooleanUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Setter
@Getter
public class RagConfig extends GlobalConfig {

    public static final String MODE_JSON = "json";

    public static final String MODE_XML = "xml";

    @JsonProperty("orchestrator")
    // Rag事前事后编排
    protected RagOrchestrator ragOrchestrator;

    @JsonProperty("remote")
    // Rag remote，使用远程服务增强内容
    protected RagRemoteConfig ragRemoteConfig;

    @JsonProperty("digest")
    // Rag digest，使用摘要记忆增强内容
    protected RagDigestConfig ragDigestConfig;

    @JsonProperty("skills")
    protected RagSkillsConfig ragSkillsConfig;

    @JsonProperty("meta")
    protected RagMetaConfig ragMetaConfig;

    @JsonProperty("mcp")
    // Rag mcp，使用MCP Resource增强内容
    protected RagMcpConfig ragMcpConfig;

    // Rag超时配置
    protected TimeoutConfig timeout;

    protected LLMConfig llmConfig;

    // 失败时是否终止整个流程
    protected Boolean stopOnFailed = false;

    // Rag Env的Key
    protected List<String> environment;

    // Rag condition，前置条件的思考链（Workflow）
    protected String condition;

    // 是否使用结果替换原始Query
    protected Boolean override;

    // Rag static query，使用静态Prompt增强内容的模板名
    protected String template;

    // 是否使用结果替换占位符内容
    protected String replace;

    // Rag remote的下游服务URL
    protected String service;

    // Rag Remote的下游服务Method
    protected String method = "POST";

    // Rag response的解析模式（XML/JSON）
    protected String mode = RagConfig.MODE_JSON;

    // Rag file资源
    protected String file;

    // 使用的Rag能力Key
    protected String key;

    // 越大排在越后，默认 10
    protected Byte sort = 10;

    public RagConfig() {

    }

    public RagConfig init(LLMConfig llmConfig) {
        this.llmConfig = llmConfig;
        return this;
    }

    public Boolean hasRagOrchestrator() {
        return this.ragOrchestrator != null;
    }

    public Boolean hasCondition() {
        return !StringUtils.isEmpty(this.condition);
    }

    public Boolean hasRagRemote() {
        return this.ragRemoteConfig != null;
    }

    public Boolean hasRagSkills() {
        return this.ragSkillsConfig != null;
    }

    public Boolean hasRagDigest() {
        return this.ragDigestConfig != null;
    }

    public Boolean hasRagMeta() {
        return this.ragMetaConfig != null;
    }

    public Boolean hasRagMcp() {
        return this.ragMcpConfig != null;
    }

    public Integer getTimeout4Condition(Integer timeout4Condition) {
        if (this.timeout == null) {
            return timeout4Condition;
        }
        return this.timeout.getTimeout4Condition(timeout4Condition);
    }

    public Integer getTimeout4Service(Integer timeout4Service) {
        if (this.timeout == null) {
            return timeout4Service;
        }
        return this.timeout.getTimeout4Service(timeout4Service);
    }

    public Integer getTimeout4Llm(Integer timeout4llm) {
        if (this.timeout == null) {
            return timeout4llm;
        }
        return this.timeout.getTimeout4Llm(timeout4llm);
    }

    public Integer getTimeout(Integer timeout) {
        if (this.timeout == null) {
            return timeout;
        }
        return this.timeout.getTimeout(timeout);
    }

    public Boolean hasReplace() {
        return !StringUtils.isEmpty(this.replace);
    }

    public Boolean isOverride() {
        return BooleanUtils.isTrue(this.override);
    }

    public Boolean isMode(String mode) {
        return StringUtils.trim(this.mode).equals(mode);
    }

    public Map<String, String> buildEnvironment() {
        Map<String, String> environment = new HashMap<String, String>();
        if (!CollectionUtils.isEmpty(this.environment)) {
            for (String key : this.environment) {
                environment.put(key, StringUtils.defaultIfBlank(System.getenv(key), System.getProperty(key)));
            }
        }
        return environment;
    }
}
