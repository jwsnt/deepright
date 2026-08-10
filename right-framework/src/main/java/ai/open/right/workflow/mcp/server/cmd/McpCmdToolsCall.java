package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.Arrays;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpCmdToolsCall extends McpCmdExportExecutor {

    @Override
    public void cmd(McpRequest mcpRequest) throws Exception {
        try {
            super.cmd(mcpRequest);
        } catch (Exception e) {
            WorkflowException.dolog(e);
            // 写入错误
            mcpRequest.write(McpCmdResponse.builder()
                    .result(ImmutableMap.of("content", Arrays.asList(ImmutableMap.of("type", "text", "text", e.getMessage())), "isError", true))
                    .build());
        }
    }

    @Override
    protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception {
        return McpCmdResponse.builder()
                .result(ImmutableMap.of("content", Arrays.asList(ImmutableMap.of("type", "text", "text", content)), "isError", false))
                .build();
    }

    @Override
    protected String buildQuery(McpRequest mcpRequest) throws Exception {
        Map<String, Object> params = Map.class.cast(mcpRequest.getContent().get("params"));
        return JsonUtils.write(Map.class.cast(params.get("arguments")));
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends McpCmdInitConfig {

        @Autowired
        @Qualifier(McpCmdConfigService.NAME_TOOLS)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_TOOLS_CALL)
        @ConditionalOnMissingBean(name = McpMethod.KEY_TOOLS_CALL)
        public McpCmdToolsCall mcpCmdToolsCall() throws Exception {
            McpCmdToolsCall mcpCmdToolsCall = new McpCmdToolsCall();
            BeanUtils.copyProperties(this, mcpCmdToolsCall);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdToolsCall inited");
            }
            return mcpCmdToolsCall;
        }
    }
}

