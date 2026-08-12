package ai.open.right.workflow.notify.impl;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;

import java.util.Map;

@Setter
@Getter
public class ShortcutNotifier {

    @Autowired
    protected NotifierService notifierService;

    @Value("${notifier.shortcut.timeout.llm:1800000}")
    // Shortcut Notifier实现类调用下游思考链（Workflow）超时
    protected Integer timeout4llm;

    public void endpoint(WorkflowTask workTask, String biz, String workflow, Map<String, Object> metadata, String content, Integer code) throws Exception {
        String[] part = SplitUtils.split(workflow, biz);
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                .content(content != null ? new StringBuffer(content) : null)
                .notifier(Notifier.ENDPOINT)
                .metadata(metadata)
                .workflow(part[1])
                .biz(part[0])
                .code(code)
                .build();
        Segment segment = Segment.build(workTask, segmentConfig);
        // Can not use chainOr2Endpoint
        this.notifierService.notify(segment, workTask, workTask);
    }

    public void endpoint(WorkflowTask workTask, String workflow, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workflow, metadata, content, code);
    }

    public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workTask.getWorkflow(), metadata, content, code);
    }

    public void endpoint(WorkflowTask workTask, String workflow, Map<String, Object> metadata, String content) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workflow, metadata, content, ProtocolCode.C200);
    }

    public void endpoint(WorkflowTask workTask, Map<String, Object> metadata, String content) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workTask.getWorkflow(), metadata, content, ProtocolCode.C200);
    }

    public void endpoint(WorkflowTask workTask, String biz, String workflow, String content, Integer code) throws Exception {
        this.endpoint(workTask, biz, workflow, workTask.getMetadata(), content, code);
    }


    public void endpoint(WorkflowTask workTask, String workflow, String content, Integer code) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workflow, workTask.getMetadata(), content, code);
    }

    public void endpoint(WorkflowTask workTask, String content, Integer code) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workTask.getWorkflow(), workTask.getMetadata(), content, code);
    }

    public void endpoint(WorkflowTask workTask, String workflow, String content) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workflow, workTask.getMetadata(), content, ProtocolCode.C200);
    }

    public void endpoint(WorkflowTask workTask, String content) throws Exception {
        this.endpoint(workTask, workTask.getBiz(), workTask.getWorkflow(), workTask.getMetadata(), content, ProtocolCode.C200);
    }

    public void source(WorkflowTask workTask, String biz, String workflow, Map<String, Object> metadata, String content, Integer code) throws Exception {
        String[] part = SplitUtils.split(workflow, biz);
        Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                .content(content != null ? new StringBuffer(content) : null)
                .notifier(Notifier.SOURCE)
                .metadata(metadata)
                .workflow(part[1])
                .biz(part[0])
                .code(code)
                .build();
        Segment segment = Segment.build(workTask, segmentConfig);
        // Can not use chainOr2Endpoint
        this.notifierService.notify(segment, workTask, workTask);
    }

    public void source(WorkflowTask workTask, String workflow, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.source(workTask, workTask.getBiz(), workflow, metadata, content, code);
    }

    public void source(WorkflowTask workTask, Map<String, Object> metadata, String content, Integer code) throws Exception {
        this.source(workTask, workTask.getBiz(), workTask.getWorkflow(), metadata, content, code);
    }

    public void source(WorkflowTask workTask, String biz, String workflow, Map<String, Object> metadata, String content) throws Exception {
        this.source(workTask, biz, workflow, metadata, content, ProtocolCode.C200);
    }

    public void source(WorkflowTask workTask, String workflow, Map<String, Object> metadata, String content) throws Exception {
        this.source(workTask, workTask.getBiz(), workflow, metadata, content, ProtocolCode.C200);
    }

    public void source(WorkflowTask workTask, Map<String, Object> metadata, String content) throws Exception {
        this.source(workTask, workTask.getBiz(), workTask.getWorkflow(), metadata, content, ProtocolCode.C200);
    }

    public void source(WorkflowTask workTask, String biz, String workflow, String content, Integer code) throws Exception {
        this.source(workTask, biz, workflow, workTask.getMetadata(), content, code);
    }

    public void source(WorkflowTask workTask, String workflow, String content, Integer code) throws Exception {
        this.source(workTask, workTask.getBiz(), workflow, workTask.getMetadata(), content, code);
    }

    public void source(WorkflowTask workTask, String content, Integer code) throws Exception {
        this.source(workTask, workTask.getBiz(), workTask.getWorkflow(), workTask.getMetadata(), content, code);
    }

    public void source(WorkflowTask workTask, String biz, String workflow, String content) throws Exception {
        this.source(workTask, biz, workflow, workTask.getMetadata(), content, ProtocolCode.C200);
    }

    public void source(WorkflowTask workTask, String workflow, String content) throws Exception {
        this.source(workTask, workTask.getBiz(), workflow, workTask.getMetadata(), content, ProtocolCode.C200);
    }

    public void source(WorkflowTask workTask, String content) throws Exception {
        this.source(workTask, workTask.getBiz(), workTask.getWorkflow(), workTask.getMetadata(), content, ProtocolCode.C200);
    }

    public SyncWorkflowTask localhost(WorkflowTask workTask, SyncConfig syncConfig) throws Exception {
        syncConfig.setTimeout(syncConfig.getTimeout() != null ? syncConfig.getTimeout() : this.timeout4llm);
        if (syncConfig.getWorkTask() == null) {
            syncConfig.setWorkTask(workTask);
        }
        return SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig);
    }
}
