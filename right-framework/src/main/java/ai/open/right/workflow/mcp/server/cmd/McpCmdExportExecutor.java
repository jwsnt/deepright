package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.context.UserContext;
import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpCmdExportService;
import ai.open.right.workflow.mcp.server.McpRequest;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

import java.util.Map;
import java.util.concurrent.Future;

@Slf4j
@Setter
@Getter
abstract public class McpCmdExportExecutor implements McpCmdExportService {

    // 不同子类不同注入
    protected McpCmdConfigService mcpCmdConfigService;

    protected RightService rightService;

    protected Integer timeout4Llm;

    // 构建查询（子类覆盖）
    abstract protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception;

    // 构建查询（子类覆盖）
    abstract protected String buildQuery(McpRequest mcpRequest) throws Exception;

    @Override
    public void export(McpExportConfig mcpExportConfig) throws Exception {
    }

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        // 检查报文
        this.checkRequest(mcpRequest);
        Assert.notNull(this.rightService, "The right service can not be empty, please config `integration.enable`");
        Future<String> future = this.rightService.get(this.buildRightConfig(mcpRequest));
        mcpRequest.write(this.buildResponse(mcpRequest, future.get()));
    }

    protected RightConfig buildRightConfig(McpRequest mcpRequest) throws Exception {
        String[] pair = this.buildDimension(mcpRequest);
        return RightConfig.builder()
                .conversation(this.buildConversation(mcpRequest))
                .userContext(this.buildUserContext(mcpRequest))
                .metadata(this.buildMetadata(mcpRequest))
                .query(this.buildQuery(mcpRequest))
                .trace(this.buildTrace(mcpRequest))
                .chat(this.buildChat(mcpRequest))
                .timeout(this.timeout4Llm)
                .workflow(pair[1]).biz(pair[0]).build();
    }

    // 构建Meta
    protected Map<String, Object> buildMetadata(McpRequest mcpRequest) throws Exception {
        return (Map<String, Object>) (Map<String, ?>) mcpRequest.getHeaders();
    }

    protected UserContext buildUserContext(McpRequest mcpRequest) throws Exception {
        return UserContext.setDefault(JsonUtils.transfer(mcpRequest.getHeaders(), UserContext.class));
    }

    protected String buildConversation(McpRequest mcpRequest) throws Exception {
        return String.class.cast(mcpRequest.getHeaders().get("conversation"));
    }

    // 构建维度
    protected String[] buildDimension(McpRequest mcpRequest) throws Exception {
        String name = String.class.cast(Map.class.cast(mcpRequest.getContent().get("params")).get("name"));
        McpExportConfig mcpExportConfig = this.mcpCmdConfigService.fetch(name);
        return new String[]{mcpExportConfig.getBiz(), mcpExportConfig.getWorkflow()};
    }

    protected String buildTrace(McpRequest mcpRequest) throws Exception {
        return StringUtils.defaultString(mcpRequest.getTrace(), String.class.cast(mcpRequest.getHeaders().get("trace")));
    }

    protected String buildChat(McpRequest mcpRequest) throws Exception {
        return String.class.cast(mcpRequest.getHeaders().get("chat"));
    }

    // 检查报文
    protected void checkRequest(McpRequest mcpRequest) throws Exception {
        Map<String, Object> params = Map.class.cast(mcpRequest.getContent().get("params"));
        Assert.notEmpty(params, "Mcp executor's params can not be empty");
        String name = String.class.cast(params.get("name"));
        Assert.hasText(name, "Mcp executor's name can not be empty");
    }

    @Setter
    @Getter
    public static class McpCmdInitConfig {

        @Autowired(required = false)
        protected RightService rightService;

        @Value("${mcp.export.timeout:1800000}")
        protected Integer timeout4Llm;
    }
}
