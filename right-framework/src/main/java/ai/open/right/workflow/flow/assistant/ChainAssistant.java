package ai.open.right.workflow.flow.assistant;

import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;

import java.util.List;
import java.util.Map;

@Setter
@Getter
@Slf4j
abstract public class ChainAssistant {

    protected NotifierService notifierService;

    protected void notify(WorkflowTask workTask, String workflow, Map<String, Object> metadata, List<MediaContext> mediaContext, String notifier, String protocol, String content, Integer code) throws Exception {
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                .content(content != null ? new StringBuffer(content) : null)
                .notifier(notifier)
                .workflow(workflow)
                .metadata(metadata)
                .protocol(protocol)
                .code(code)
                .build();
        Segment segment = Segment.build(workTask, segmentConfig);
        this.notifierService.notify(segment, workTask, workTask, mediaContext);
    }

    protected void notify(WorkflowTask workTask, String workflow, Map<String, Object> metadata, String notifier, String content) throws Exception {
        this.notify(workTask, workflow, metadata, null, notifier, Protocol.CHAT, content, ProtocolCode.C200);
    }

    protected void notify(WorkflowTask workTask, Map<String, Object> metadata, String notifier, String content) throws Exception {
        this.notify(workTask, workTask.getWorkflow(), metadata, null, notifier, Protocol.CHAT, content, ProtocolCode.C200);
    }

    protected void notify(WorkflowTask workTask, String workflow, String notifier, String content) throws Exception {
        this.notify(workTask, workflow, null, null, notifier, Protocol.CHAT, content, ProtocolCode.C200);
    }

    protected void notify(WorkflowTask workTask, String notifier, String content) throws Exception {
        this.notify(workTask, workTask.getWorkflow(), null, null, notifier, Protocol.CHAT, content, ProtocolCode.C200);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Chain or endpoint: metadata={}, media={}, protocol={}, content={}, code={}", metadata, mediaContext, protocol, content, code);
        }
        if (workflowConfig.hasChain()) {
            this.notify(workTask, workflowConfig.getChain(), metadata, mediaContext, workflowConfig.getNotifier(Notifier.LOCALHOST), Protocol.CHAT, content, code);
        } else {
            // 未指定 或 非Chain时Localhost 修改为ENDPOINT
            String notifier = StringUtils.defaultIfBlank(!StringUtils.equalsIgnoreCase(workTask.getNotifier(), Notifier.LOCALHOST) ? workTask.getNotifier() : Notifier.ENDPOINT, Notifier.ENDPOINT);
            this.notify(workTask, null, metadata, mediaContext, notifier, Protocol.CHAT, content, code);
        }
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, String protocol, String content, Integer code) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, metadata, null, protocol, content, code);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, String protocol, String content) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, metadata, null, protocol, content, ProtocolCode.C200);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, mediaContext, protocol, content, code);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content, Integer code) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, null, protocol, content, code);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, List<MediaContext> mediaContext, String protocol, String content) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, mediaContext, protocol, content, ProtocolCode.C200);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, null, protocol, content, ProtocolCode.C200);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, List<MediaContext> mediaContext, String content, Integer code) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, mediaContext, Protocol.CHAT, content, code);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content, Integer code) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, null, Protocol.CHAT, content, code);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, List<MediaContext> mediaContext, String content) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, mediaContext, Protocol.CHAT, content, ProtocolCode.C200);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, metadata, null, Protocol.CHAT, content, code);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, String content) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, metadata, null, Protocol.CHAT, content, ProtocolCode.C200);
    }

    public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String content) throws Exception {
        this.chainOr2Endpoint(workflowConfig, workTask, null, null, Protocol.CHAT, content, ProtocolCode.C200);
    }

    @Setter
    @Getter
    public static class ChainInitConfig {

        @Autowired
        protected NotifierService notifierService;
    }
}
