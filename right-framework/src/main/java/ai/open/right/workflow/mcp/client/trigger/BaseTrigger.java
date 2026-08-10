package ai.open.right.workflow.mcp.client.trigger;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.impl.ShortcutNotifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;

import java.util.Map;

@Slf4j
@Setter
@Getter
public class BaseTrigger extends ShortcutNotifier implements McpTrigger {

    @Autowired
    protected NotifierService notifierService;

    @Override
    public void beforeToolsCall(McpDimension mcpDimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Trigger tools call");
        }
    }

    @Override
    public void beforePromptGet(McpDimension mcpDimension, Map<String, Object> arguments, WorkflowTask workTask) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Trigger prompt get");
        }
    }

    @Override
    public void beforeResourcesRead(McpDimension mcpDimension, String uri, WorkflowTask workTask) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Trigger resources read");
        }
    }

    public void endpoint(McpContext<?> context, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.endpoint(context.getWorkTask(), metadata, content, code);
    }

    public void endpoint(McpContext<?> context, Map<String, Object> metadata, String content) throws Exception {
        this.endpoint(context.getWorkTask(), metadata, content, ProtocolCode.C200);
    }

    public void endpoint(McpContext<?> context, String content, Integer code) throws Exception {
        this.endpoint(context.getWorkTask(), context.getWorkTask().getMetadata(), content, code);
    }

    public void endpoint(McpContext<?> context, String content) throws Exception {
        this.endpoint(context.getWorkTask(), context.getWorkTask().getMetadata(), content, ProtocolCode.C200);
    }

    public void source(McpContext<?> context, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.source(context.getWorkTask(), metadata, content, code);
    }

    public void source(McpContext<?> context, Map<String, Object> metadata, String content) throws Exception {
        this.source(context.getWorkTask(), metadata, content, ProtocolCode.C200);
    }

    public void source(McpContext<?> context, String content, Integer code) throws Exception {
        this.source(context.getWorkTask(), context.getWorkTask().getMetadata(), content, code);
    }

    public void source(McpContext<?> context, String content) throws Exception {
        this.source(context.getWorkTask(), context.getWorkTask().getMetadata(), content, ProtocolCode.C200);
    }

    public SyncWorkflowTask localhost(McpContext<?> context, SyncConfig syncConfig) throws Exception {
        return this.localhost(context.getWorkTask(), syncConfig);
    }
}
