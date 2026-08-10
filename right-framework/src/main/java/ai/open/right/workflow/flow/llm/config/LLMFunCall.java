package ai.open.right.workflow.flow.llm.config;

import ai.open.right.utils.CollectionsUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.config.AllowedConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.List;
import java.util.Map;

@ToString
@Setter
@Getter
@Slf4j
// Fun Call配置
public class LLMFunCall extends AllowedConfig implements ProviderFunCall {

    // MCP型Fun Call时，静态替换指定Fun Call的Description
    protected Map<String, String> descriptions;

    // MCP型Fun Call时，静态替换指定Fun Call的Properties
    protected Map<String, Object> properties;

    // 思考链（Workflow）型Fun Call的Required属性
    protected List<String> required;

    // 接管思考链（Workflow）型Fun Call的
    protected LLMTakeover takeover;

    // 思考链（Workflow）型Fun Call的Description属性
    protected String description;

    // MCP型Fun Call时，Description的前缀
    protected String prefix;

    // MCP型Fun Call时，Description的后缀
    protected String suffix;

    // 当Refer=True表示当前Fun Call为MCP型，会从MCP配置加载
    protected Boolean refer;

    // 思考链（Workflow）型Fun Call的Name属性
    protected String name;

    public LLMFunCall merge(LLMFunCall llmFunCall) throws Exception {
        super.merge(llmFunCall);
        if (llmFunCall != null) {
            // 入参覆盖当前值
            this.takeover = this.takeover != null ? this.takeover.merge(llmFunCall.takeover) : llmFunCall.takeover;
            this.description = StringUtils.defaultIfBlank(this.description, llmFunCall.description);
            this.descriptions = CollectionsUtils.merge(this.descriptions, llmFunCall.descriptions);
            this.properties = CollectionsUtils.merge(this.properties, llmFunCall.properties);
            this.whiteList = CollectionsUtils.merge(this.whiteList, llmFunCall.whiteList);
            this.blackList = CollectionsUtils.merge(this.blackList, llmFunCall.blackList);
            this.required = CollectionsUtils.merge(this.required, llmFunCall.required);
            this.prefix = StringUtils.defaultIfBlank(this.prefix, llmFunCall.prefix);
            this.suffix = StringUtils.defaultIfBlank(this.suffix, llmFunCall.suffix);
            this.name = StringUtils.defaultIfBlank(this.name, llmFunCall.name);
            this.refer = llmFunCall.refer;
        }
        return this;
    }

    public LLMFunCall init(String notifier) {
        if (this.hasTakeover()) {
            this.getTakeover().init(notifier);
        }
        return this;
    }

    public Boolean getRefer() {
        return this.refer != null ? this.refer : false;
    }

    public Boolean hasDescriptions(String key) {
        return !CollectionUtils.isEmpty(this.descriptions) && this.descriptions.containsKey(key);
    }

    public Boolean hasProperties(String key) {
        return !CollectionUtils.isEmpty(this.properties) && this.properties.containsKey(key);
    }

    public Boolean hasPrefix() {
        return !StringUtils.isEmpty(this.getPrefix());
    }

    public Boolean hasSuffix() {
        return !StringUtils.isEmpty(this.getSuffix());
    }

    public Boolean hasTakeover() {
        return this.takeover != null;
    }

    public Boolean isLooped(String biz, String workflow) {
        String name = SplitUtils.join(workflow, biz);
        // 全匹配 或 者尾部匹配
        return StringUtils.equalsIgnoreCase(this.getName(), name) || StringUtils.endsWithIgnoreCase(name, this.getName());
    }
}
