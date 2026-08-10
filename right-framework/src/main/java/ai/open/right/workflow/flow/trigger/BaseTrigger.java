package ai.open.right.workflow.flow.trigger;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.impl.ShortcutNotifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;

import java.util.Map;

@Setter
@Getter
@Slf4j
public class BaseTrigger extends ShortcutNotifier implements WorkflowTrigger {

    @Autowired
    protected NotifierService notifierService;

    @Override
    public void before(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Trigger workflow");
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
