package ai.open.right.workflow.flow.assistant;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.command.QuickCommandStore;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import ai.open.right.workflow.flow.tools.ToolsPackage;
import ai.open.right.workflow.flow.tools.ToolsResponse;
import ai.open.right.workflow.flow.tools.ToolsService;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.exc.MismatchedInputException;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

@Slf4j
@Setter
@Getter
// 自定义工具
public class ToolsAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-tools";

    protected QuickCommandStore quickCommandStore;

    protected ToolsService toolsService;

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        Assert.isTrue(workflowConfig.hasTools(), "Tools config can not be empty, please check config");
        String response = this.toolsService.execute(workflowConfig.getToolsConfig(), workTask);
        this.doingToolsResponse(workflowConfig, workTask, response);
    }

    protected void doingToolsResponse(WorkflowConfig workConfig, WorkflowTask workTask, String response) throws Exception {
        try {
            if (log.isInfoEnabled()) {
                log.info("Prepare to convert to the json format of ToolsResponse={}", response);
            }
            ToolsResponse toolsResponse = JsonUtils.read(response, ToolsResponse.class);
            if (toolsResponse == null || toolsResponse.getCode() == null) {
                // 无法解析或识别
                String content = this.buildContent(workConfig.getToolsConfig(), response, workTask.getQuery());
                this.chainOr2Endpoint(workConfig, workTask, Protocol.TOOL, content);
                return;
            }
            // 解析ToolsResponse状态, Default code 500
            Integer code = toolsResponse.getCode(ProtocolCode.C500);
            // 如果配置Tools Config则精确检查状态码，否则检查是否2xx
            if (!(workConfig.hasTools() ? workConfig.getToolsConfig().isSuccessCode(code) : ProtocolCode.range2xx(code))) {
                throw new WorkflowException(toolsResponse.getMsg(response), code);
            }
            String content = null;
            // 处理解析后的Data
            if (toolsResponse.hasData()) {
                // 处理Quick Command
                this.storeQuickCommand(workTask, workConfig.getToolsConfig(), toolsResponse);
                // 重置真实Content
                content = toolsResponse.getData().getContent();
            }
            // 解析出报文，或报文和响应均为空
            if (!StringUtils.isEmpty(content) || (StringUtils.isEmpty(content) && StringUtils.isEmpty(response))) {
                // Add metadata
                content = this.buildContent(workConfig.getToolsConfig(), content, workTask.getInitial());
                this.chainOr2Endpoint(workConfig, workTask, toolsResponse.getMetadata(), Protocol.TOOL, content);
            }else {
                this.chainOr2Endpoint(workConfig, workTask, toolsResponse.getMetadata(), Protocol.TOOL, response);
            }
        } catch (JsonParseException | MismatchedInputException e) {
            if (log.isDebugEnabled()) {
                log.debug("Json parse failed={}-{}", e.getMessage(), workTask.getQuery());
            }
            // 解析失败则传递原始响应
            this.chainOr2Endpoint(workConfig, workTask, Protocol.TOOL, response);
        }
    }

    protected void storeQuickCommand(WorkflowTask workTask, ToolsConfig toolsConfig, ToolsResponse toolsResponse) {
        if (!CollectionUtils.isEmpty(toolsResponse.getData().getCommand()) && this.quickCommandStore != null) {
            // 存储Tools快捷指令并推送
            this.quickCommandStore.store(toolsResponse.getData().getCommand(), toolsConfig != null ? toolsConfig.getExpired() : null, workTask.getBiz(), workTask.getChat(), workTask.getUserContext().getDevice());
        }
    }

    protected String buildContent(ToolsConfig toolsConfig, Object response, String query) throws Exception {
        // Wrap Query or not
        // Source=True表示包装思考链（Workflow）初始（Initial）Query
        return JsonUtils.write(toolsConfig != null ? (toolsConfig.shouldSource() ? ToolsPackage.builder()
                .query(query)
                .tools(response)
                .build() : response) : response);
    }

    @ConditionalOnProperty(name = "tools.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired(required = false)
        protected QuickCommandStore quickCommandStore;

        @Autowired
        protected ToolsService toolsService;

        @Bean(ToolsAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ToolsAssistant.WORKFLOW_NAME)
        public ToolsAssistant toolsAssistant() throws Exception {
            ToolsAssistant toolsAssistant = new ToolsAssistant();
            BeanUtils.copyProperties(this, toolsAssistant);
            log.info("ToolsAssistant inited");
            return toolsAssistant;
        }
    }
}
