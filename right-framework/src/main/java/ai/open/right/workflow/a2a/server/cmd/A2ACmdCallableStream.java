package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2AData;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.*;
import ai.open.right.workflow.flow.llm.Segment;
import com.fasterxml.jackson.core.JsonParseException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.Assert;

import java.util.List;
import java.util.UUID;

@Slf4j
public class A2ACmdCallableStream extends A2ACmdCallableOnce {

    public A2ACmdCallableStream(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        super(a2aRequest, messageRequest);
    }

    @Override
    public void call(Segment segment) throws Exception {
        try {
            if (ProtocolCode.range2xx(segment.getCode())) {
                // 正确返回
                TaskArtifactUpdateEvent taskArtifactUpdateEvent = this.buildTaskArtifactUpdateEvent(segment);
                A2ACmdResponse response = this.buildA2ACmdResponse(segment, taskArtifactUpdateEvent);
                this.write(response);
                if (segment.isFinished()) {
                    // 如果Stream结束
                    this.close(TaskStatus.STATUS_COMPLETED);
                }
            } else {
                // Code非2xx，关闭流
                this.close(TaskStatus.STATUS_FAILED);
            }
        } catch (Exception e) {
            // 异常，关闭流
            WorkflowException.dolog(e);
            this.close(TaskStatus.STATUS_FAILED);
        }
    }

    // 构建A2A Response
    protected A2ACmdResponse buildA2ACmdResponse(Segment segment, TaskArtifactUpdateEvent taskArtifactUpdateEvent) throws Exception {
        return A2ACmdResponse.builder()
                .result(taskArtifactUpdateEvent.reset())
                .finished(segment.isFinished())
                .id(this.buildA2AId())
                .build();
    }

    // 构建Task Event
    protected TaskArtifactUpdateEvent buildTaskArtifactUpdateEvent(Segment segment) throws Exception {
        Artifact artifact = null;
        try {
            // 转换为A2AData（兼容Object/TaskArtifactUpdateEvent/Artifact）
            A2AData a2aData = this.buildA2AData(segment);
            if (a2aData.isSupport(TaskArtifactUpdateEvent.PROTOCOL)) {
                // 直接转为Task，清除标记
                if (log.isDebugEnabled()) {
                    log.debug("A2AData is converted to TaskArtifactUpdateEvent");
                }
                // TaskArtifactUpdateEvent直接返回
                return JsonUtils.transfer(a2aData, TaskArtifactUpdateEvent.class)
                        .metadata(this.buildTaskMetadata(segment))
                        .contextId(this.buildContextId())
                        .lastChunk(segment.isFinished())
                        .append(!segment.isFinished())
                        .taskId(this.buildTaskId());
            }
            if (a2aData.isSupport(Artifact.PROTOCOL)) {
                if (log.isDebugEnabled()) {
                    log.debug("A2AData is converted to Artifact");
                }
                // 直接转为Artifact，清除标记
                artifact = JsonUtils.transfer(a2aData, Artifact.class);
            } else {
                if (log.isDebugEnabled()) {
                    log.debug("A2AData is converted to Map Data");
                }
                // 构建A2AData（Map）的Artifact
                artifact = this.buildDataArtifact(a2aData);
            }
        } catch (JsonParseException e) {
            // 无法解析Json，通常为[]，直接使用String构建Artifact
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            artifact = this.buildTextArtifact(segment);
        }
        Assert.notNull(artifact, "Artifact can not be empty: " + segment.getContent());
        return TaskArtifactUpdateEvent.builder()
                // 填充属性
                .artifact(artifact.artifactId(String.valueOf(segment.getIndex()))
                        .metadata(segment.getMetadata()))
                .metadata(this.buildTaskMetadata(segment))
                .contextId(this.buildContextId())
                .lastChunk(segment.isFinished())
                .append(!segment.isFinished())
                .taskId(this.buildTaskId())
                .build();
    }

    protected A2AData buildA2AData(Segment segment) throws Exception {
        if (!JsonUtils.like(segment.getContent())) {
            // Simple Data Part
            A2AData a2AData = JsonUtils.transfer(this.buildArtifact(segment, this.buildPart(segment)), A2AData.class);
            a2AData.put("internal", Artifact.PROTOCOL);
            return a2AData;
        } else {
            return JsonUtils.read(segment.getContent(), A2AData.class)
                    .bindSegment(segment);
        }
    }

    protected Artifact buildArtifact(Segment segment, Part part) throws Exception {
        return Artifact.builder()
                .artifactId(UUID.randomUUID().toString())
                .parts(List.of(part))
                .build()
                .reset();
    }

    protected Part buildPart(Segment segment) throws Exception {
        return Part.builder()
                .metadata(segment.getMetadata())
                .text(segment.getContent())
                .build();
    }

    // 写入流
    protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
        this.a2aRequest.writeStream(a2ACmdResponse);
    }

    // 流关闭
    protected void close(String status) throws Exception {
        TaskStatusUpdateEvent task = TaskStatusUpdateEvent.builder()
                .contextId(this.buildContextId())
                .status(TaskStatus.builder()
                        .state(status)
                        .build())
                .taskId(this.buildTaskId())
                .finished(true)
                .build();
        this.write(A2ACmdResponse.builder()
                .id(this.buildTaskId())
                // 标记SEE结束
                .finished(true)
                .result(task)
                .build());
    }

    // 发送流前
    protected void start() throws Exception {
        Task task = Task.builder()
                .timestamp(this.buildTimestamp(System.currentTimeMillis()))
                .contextId(this.buildContextId())
                .status(TaskStatus.builder()
                        .state(TaskStatus.STATUS_SUBMITTED)
                        .build())
                .id(this.buildTaskId())
                .build();
        this.write(A2ACmdResponse.builder()
                .id(this.buildTaskId())
                .result(task.reset())
                .build());
    }
}
