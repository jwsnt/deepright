package ai.open.right.workflow.mcp.client.rewrtier;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpContext;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.impl.ShortcutNotifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;

import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
public class BaseRewriter extends ShortcutNotifier implements McpRewriter {

    @Autowired
    protected NotifierService notifierService;

    @Override
    public McpResult<List<Map<String, Object>>> toolsCall(McpContext<List<Map<String, Object>>> context) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Rewrite tools call={}", context);
        }
        return context.getResult();
    }

    public McpResult<List<History>> promptGet(McpContext<List<History>> context) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Rewrite prompt get={}", context);
        }
        return context.getResult();
    }

    public McpResult<String> resourcesRead(McpContext<String> context) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Rewrite resource read={}", context);
        }
        return context.getResult();
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
