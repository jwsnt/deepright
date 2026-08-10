package ai.open.right.workflow.flow.function.impl;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.function.Function;
import ai.open.right.workflow.flow.function.FunctionContext;
import ai.open.right.workflow.notify.impl.ShortcutNotifier;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;

@Setter
@Getter
public class BaseFunction extends ShortcutNotifier implements Function {

    public void endpoint(FunctionContext functionContext, Map<String, Object> metadata, String content, Integer code) throws Exception {
        super.endpoint(functionContext.getWorkTask(), metadata, content, code);
    }

    public void endpoint(FunctionContext functionContext, Map<String, Object> metadata, String content) throws Exception {
        super.endpoint(functionContext.getWorkTask(), metadata, content, ProtocolCode.C200);
    }

    public void endpoint(FunctionContext functionContext, String content, Integer code) throws Exception {
        super.endpoint(functionContext.getWorkTask(), functionContext.getWorkTask().getMetadata(), content, code);
    }

    public void endpoint(FunctionContext functionContext, String content) throws Exception {
        super.endpoint(functionContext.getWorkTask(), functionContext.getWorkTask().getMetadata(), content, ProtocolCode.C200);
    }

    public void source(FunctionContext functionContext, Map<String, Object> metadata, String content, Integer code) throws Exception {
        super.source(functionContext.getWorkTask(), metadata, content, code);
    }

    public void source(FunctionContext functionContext, Map<String, Object> metadata, String content) throws Exception {
        super.source(functionContext.getWorkTask(), metadata, content, ProtocolCode.C200);
    }

    public void source(FunctionContext functionContext, String content, Integer code) throws Exception {
        super.source(functionContext.getWorkTask(), functionContext.getWorkTask().getMetadata(), content, code);
    }

    public void source(FunctionContext functionContext, String content) throws Exception {
        super.source(functionContext.getWorkTask(), functionContext.getWorkTask().getMetadata(), content, ProtocolCode.C200);
    }

    public SyncWorkflowTask localhost(FunctionContext functionContext, SyncConfig syncConfig) throws Exception {
        return super.localhost(functionContext.getWorkTask(), syncConfig);
    }

    public Object call(FunctionContext functionContext) throws Exception {
        return functionContext.getWorkTask().getQuery();
    }
}
