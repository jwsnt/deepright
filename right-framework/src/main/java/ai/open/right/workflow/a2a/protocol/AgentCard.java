package ai.open.right.workflow.a2a.protocol;

import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

import java.util.List;

// 服务发现URL：https://{server_domain}/.well-known/agent-card.json
@Setter
@Getter
public class AgentCard {

    public static final List<String> MODELS = List.of("application/json");

    // 代理支持的可选功能声明
    private AgentCapabilities capabilities = new AgentCapabilities();

    // 所有技能支持的默认输出MIME类型集，可在每个技能基础上覆盖
    protected List<String> defaultOutputModes = AgentCard.MODELS;

    // 所有技能支持的默认输入MIME类型集，可在每个技能基础上覆盖
    protected List<String> defaultInputModes = AgentCard.MODELS;

    // 端点的传输协议，默认为"JSONRPC"，必须存在于每个中AgentCard
    protected String preferredTransport = "JSONRPC";

    // 支持的A2A协议版本，默认"0.3.0"
    protected String protocolVersion = "0.3.0";

    // 可读描述，帮助用户和其他代理理解其目的
    protected String description = "a2a server";

    // 代理可以执行的技能或不同能力的集合
    protected List<AgentSkill> skills;

    // 代理自己的版本号，格式由提供者定义
    protected String version = "1.0";

    // 可读名称，BIZ@Workflow
    protected String name;

    // 与代理交互的首选端点URL，必须支持preferredTransport指定的传输
    protected String url;

    // 用于配置覆盖
    public void setDescription(String description) {
        if (!StringUtils.isEmpty(description)) {
            this.description = description;
        }
    }

    public Boolean hasSkill() {
        return !CollectionUtils.isEmpty(this.skills);
    }
}
    