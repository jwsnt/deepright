package ai.open.right.workflow.mcp.server.cmd;

import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.config.McpExportConfig;
import ai.open.right.workflow.mcp.McpMethod;
import ai.open.right.workflow.mcp.server.McpCmdConfigService;
import ai.open.right.workflow.mcp.server.McpRequest;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class McpCmdResourcesRead extends McpCmdExportExecutor {

    @Override
    protected McpCmdResponse buildResponse(McpRequest mcpRequest, String content) throws Exception {
        return McpCmdResponse.builder()
                .result(ImmutableMap.of("contents", Arrays.asList(ImmutableMap.of("uri", "", "mimeType", "text/plain", "text", content))))
                .build();
    }

    @Override
    // 构建查询, 如果配置了Query则使用，否则使用Name
    protected String buildQuery(McpRequest mcpRequest) throws Exception {
        String[] buildDimension = this.buildDimension(mcpRequest);
        if (buildDimension.length > 2) {
            // Resources Templates
            StringBuffer request = new StringBuffer();
            for (int index = 2; index < buildDimension.length; index++) {
                request.append(URLDecoder.decode(JsonUtils.clean(buildDimension[index]), StandardCharsets.UTF_8)).append(SplitUtils.SPLIT_SLASH);
            }
            return StringUtils.substring(request.toString(), 0, request.toString().length() - 1);
        } else {
            // Resources
            McpExportConfig mcpExportConfig = this.mcpCmdConfigService.fetch(String.class.cast(Map.class.cast(mcpRequest.getContent().get("params")).get("uri")));
            String query = mcpExportConfig.hasQuery() ? mcpExportConfig.getQuery() : mcpExportConfig.getName();
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources read query={}", query);
            }
            return query;
        }
    }

    @Override
    // 构建维度
    protected String[] buildDimension(McpRequest mcpRequest) throws Exception {
        String name = String.class.cast(Map.class.cast(mcpRequest.getContent().get("params")).get("uri"));
        String[] part = SplitUtils.splitWithSymbol(SplitUtils.SPLIT_SLASH, name);
        // Resources由2段构成，ResourcesTemplates由大于2段构成
        if (part.length > 2) {
            // Resources Templates
            McpExportConfig mcpExportConfig = this.mcpCmdConfigService.fetch(McpExportConfig.buildTemplateFormat(SplitUtils.join(SplitUtils.SPLIT_SLASH, part[1], part[0])));
            part[1] = mcpExportConfig.getWorkflow();
            part[0] = mcpExportConfig.getBiz();
            return part;
        } else {
            // Resource
            McpExportConfig mcpExportConfig = this.mcpCmdConfigService.fetch(name);
            return new String[]{mcpExportConfig.getBiz(), mcpExportConfig.getWorkflow()};
        }
    }

    @Override
    protected void checkRequest(McpRequest mcpRequest) throws Exception {
        Map<String, Object> params = Map.class.cast(mcpRequest.getContent().get("params"));
        Assert.notEmpty(params, "Mcp executor's params can not be empty");
        String uri = String.class.cast(params.get("uri"));
        Assert.hasText(uri, "Mcp executor's uri can not be empty");
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends McpCmdInitConfig {

        @Autowired
        @Qualifier(McpCmdConfigService.NAME_RESOURCES)
        protected McpCmdConfigService mcpCmdConfigService;

        @Bean(McpMethod.KEY_RESOURCES_READ)
        @ConditionalOnMissingBean(name = McpMethod.KEY_RESOURCES_READ)
        public McpCmdResourcesRead mcpCmdResourcesRead() throws Exception {
            McpCmdResourcesRead mcpCmdResourcesRead = new McpCmdResourcesRead();
            BeanUtils.copyProperties(this, mcpCmdResourcesRead);
            if (log.isDebugEnabled()) {
                log.debug("McpCmdResourcesRead inited");
            }
            return mcpCmdResourcesRead;
        }
    }
}