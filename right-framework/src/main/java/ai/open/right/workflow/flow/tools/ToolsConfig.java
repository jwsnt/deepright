package ai.open.right.workflow.flow.tools;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.CollectionsUtils;
import ai.open.right.workflow.flow.config.GlobalConfig;
import ai.open.right.workflow.flow.config.TimeoutConfig;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.BooleanUtils;
import org.springframework.util.CollectionUtils;
import org.springframework.util.StringUtils;

import java.util.List;

@Setter
@Getter
public class ToolsConfig extends GlobalConfig {

    public static final String WRAP_OBJECT = "object";

    public static final String WRAP_STRING = "string";

    public static final String WRAP_SOURCE = "source";

    @JsonProperty("orchestrator")
    // Tools请求/响应编排
    protected ToolsOrchestrator toolsOrchestrator;

    // 追加Header
    protected List<ToolsHeader> headers;

    // 超时配置
    protected TimeoutConfig timeout;

    // 默认Success Code
    protected Integer successCode;

    // Quick Command持久化时间
    protected Integer expired;

    // 下游服务地址
    protected String service;

    // Source=True表示在响应中包装原始Query
    protected Boolean source;

    // 下游服务请求类型
    protected String method;

    // 请求包装类型（WRAP_OBJECT/WRAP_STRING/WRAP_SOURCE）
    protected String wrap;

    public ToolsConfig merge(ToolsConfig toolsConfig) throws Exception {
        super.merge(toolsConfig);
        if (toolsConfig != null) {
            this.toolsOrchestrator = this.toolsOrchestrator != null ? this.toolsOrchestrator : toolsConfig.toolsOrchestrator;
            this.timeout = this.timeout != null ? this.timeout.merge(toolsConfig.timeout) : toolsConfig.timeout;
            this.successCode = this.successCode != null ? this.successCode : toolsConfig.successCode;
            this.service = StringUtils.hasText(this.service) ? this.service : toolsConfig.service;
            this.method = StringUtils.hasText(this.method) ? this.method : toolsConfig.method;
            this.wrap = StringUtils.hasText(this.wrap) ? this.wrap : toolsConfig.wrap;
            this.headers = CollectionsUtils.merge(this.headers, toolsConfig.headers);
            this.expired = this.expired != null ? this.expired : toolsConfig.expired;
            this.source = this.source != null ? this.source : toolsConfig.source;
        }
        return this;
    }

    public Boolean hasOrchestrator() {
        return this.toolsOrchestrator != null;
    }

    public Boolean hasHeaders() {
        return !CollectionUtils.isEmpty(this.headers);
    }

    public Integer getTimeout4Service(Integer timeout4Service) {
        if (this.timeout == null) {
            return timeout4Service;
        }
        return this.timeout.getTimeout4Service(timeout4Service);
    }

    public Integer getTimeout4Llm(Integer timeout4Llm) {
        if (this.timeout == null) {
            return timeout4Llm;
        }
        return this.timeout.getTimeout4Llm(timeout4Llm);
    }

    public Integer getSuccessCode() {
        return this.successCode != null ? this.successCode : ProtocolCode.C200;
    }

    public String getMethod() {
        return this.method != null ? this.method : "POST";
    }

    public Boolean shouldSource() {
        return BooleanUtils.isTrue(this.source);
    }

    public Boolean shouldWrap() {
        return StringUtils.hasText(this.wrap);
    }

    public Boolean isSuccessCode(Integer code) {
        return this.getSuccessCode().equals(code);
    }

    public Boolean isValidWrap() {
        return ToolsConfig.WRAP_OBJECT.equals(this.wrap) || ToolsConfig.WRAP_STRING.equals(this.wrap) || ToolsConfig.WRAP_SOURCE.equals(this.wrap);
    }
}
