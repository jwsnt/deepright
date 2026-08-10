package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.fasterxml.jackson.core.JacksonException;
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

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpCmdPromptGet extends McpCmdExportExecutor {

    @Override
    protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception {
        try {
            // 尝试解析为多个McpCmdPrompt
            McpCmdPrompt[] prompts = JsonUtils.read(content, McpCmdPrompt[].class);
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt get={}", Arrays.toString(prompts));
            }
            // 构建多个Prompt
            List<Map<String, Object>> messages = new ArrayList<Map<String, Object>>();
            for (McpCmdPrompt prompt : prompts) {
                messages.add(ImmutableMap.of("role", prompt.getRole(), "content", ImmutableMap.of("type", "text", "text", prompt.getContent())));
            }
            return McpCmdResponse.builder()
                    .result(ImmutableMap.of("messages", messages))
                    .build();
        } catch (JacksonException e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            // 构建单个Prompt
            return McpCmdResponse.builder()
                    .result(ImmutableMap.of("messages", Arrays.asList(ImmutableMap.of("role", "user", "content", ImmutableMap.of("type", "text", "text", content)))))
                    .build();
        }
    }

    // 构建查询, 如果配置了Query则使用，否则使用Name
    protected String buildQuery(McpRequest mcpRequest) throws Exception {
        Map<String, Object> params = Map.class.cast(mcpRequest.getContent().get("params"));
        McpExportConfig mcpExportConfig = this.mcpCmdConfigService.fetch(String.class.cast(params.get("name")));
        String query = mcpExportConfig.hasQuery() ? mcpExportConfig.getQuery() : mcpExportConfig.getName();
        if (log.isDebugEnabled()) {
            log.debug("Mcp prompt get query={}", query);
        }
        return query;
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends McpCmdInitConfig {

        @Autowired
        @Qualifier(McpCmdConfigService.NAME_PROMPTS)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_PROMPTS_GET)
        @ConditionalOnMissingBean(name = McpMethod.KEY_PROMPTS_GET)
        public McpCmdPromptGet mcpCmdPromptGet() throws Exception {
            McpCmdPromptGet mcpCmdPromptGet = new McpCmdPromptGet();
            BeanUtils.copyProperties(this, mcpCmdPromptGet);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdPromptGet inited");
            }
            return mcpCmdPromptGet;
        }
    }
}