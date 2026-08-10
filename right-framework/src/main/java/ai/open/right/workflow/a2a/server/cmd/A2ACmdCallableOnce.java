package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2AData;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.*;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.sync.impl.BaseCallable;
import com.fasterxml.jackson.core.JsonParseException;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.time.FastDateFormat;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;
import java.util.TimeZone;
import java.util.UUID;

@Slf4j
@Setter
@Getter
// Message/Send回调
public class A2ACmdCallableOnce extends BaseCallable {

    // 统一时区的时间
    protected static final FastDateFormat FORMATTER = FastDateFormat.getInstance("yyyy-MM-dd'T'HH:mm:ss'Z'", TimeZone.getTimeZone("UTC"));

    protected final MessageRequest messageRequest;

    protected final A2ARequest a2aRequest;

    public A2ACmdCallableOnce(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        this.messageRequest = messageRequest;
        this.a2aRequest = a2aRequest;
        // 等待回调前调用
        this.start();
    }

    @Override
    public void call(Segment segment) throws Exception {
        try {
            // 根据状态码构建Task（2xx构建Success，非2xx构建Failed）
            Task task = ProtocolCode.range2xx(segment.getCode()) ? this.buildSuccessTask(segment) : this.buildFailedTask(segment);
            A2ACmdResponse response = this.buildA2ACmdResponse(segment, task);
            this.write(response);
        } catch (Exception e) {
            // 异常，构建错误消息并关闭
            WorkflowException.dolog(e);
            this.close(e.getMessage(), TaskStatus.STATUS_FAILED);
        }
    }

    // 构建A2A Response
    protected A2ACmdResponse buildA2ACmdResponse(Segment segment, Task task) throws Exception {
        return A2ACmdResponse.builder()
                .finished(segment.isFinished())
                .id(this.buildA2AId())
                .result(task.reset())
                .build();
    }

    // 构建Task的Meta
    protected Map<String, Object> buildTaskMetadata(Segment segment) throws Exception {
        return segment.getMetadata();
    }

    // 使用A2AData（Map）构建Artifact
    protected Artifact buildDataArtifact(A2AData a2aData) throws Exception {
        return Artifact.builder()
                .parts(List.of(Part.builder()
                        .kind(Part.DATA_KIND)
                        .data(a2aData)
                        .build()))
                .build().reset();
    }

    // 使用String构建Artifact
    protected Artifact buildTextArtifact(Segment segment) throws Exception {
        return Artifact.builder()
                .parts(List.of(Part.builder()
                        .text(segment.getContent())
                        .kind(Part.TEXT_KIND)
                        .build()))
                .build().reset();
    }

    // 使用String构建Artifact，Artifact ID = UUID
    protected Artifact buildTextArtifact(String content) throws Exception {
        return Artifact.builder()
                .parts(List.of(Part.builder()
                        .kind(Part.TEXT_KIND)
                        .text(content)
                        .build()))
                // 使用UUID
                .artifactId(UUID.randomUUID().toString())
                .build().reset();
    }

    // 构建Task的状态
    protected TaskStatus buildTaskStatus(Segment segment) throws Exception {
        String status = ProtocolCode.range2xx(segment.getCode()) ? TaskStatus.STATUS_COMPLETED : TaskStatus.STATUS_FAILED;
        return TaskStatus.builder()
                .state(status)
                .build();
    }

    // 构建时间戳
    protected String buildTimestamp(Segment segment) throws Exception {
        return this.buildTimestamp(segment.getTimestamp());
    }

    protected String buildTimestamp(Long timestamp) throws Exception {
        return A2ACmdCallableOnce.FORMATTER.format(timestamp);
    }

    // 构建成功的Task
    protected Task buildSuccessTask(Segment segment) throws Exception {
        Artifact artifact = null;
        try {
            // 转换为A2AData（兼容Object/Task/Artifact）
            A2AData a2aData = JsonUtils.read(segment.getContent(), A2AData.class)
                    .bindSegment(segment);
            if (a2aData.isSupport(Task.PROTOCOL)) {
                // 直接转为Task，清除标记
                if (log.isDebugEnabled()) {
                    log.debug("A2AData is converted to Task");
                }
                // 填充属性后直接返回
                return JsonUtils.transfer(a2aData, Task.class)
                        .metadata(this.buildTaskMetadata(segment))
                        .timestamp(this.buildTimestamp(segment))
                        .status(this.buildTaskStatus(segment))
                        .contextId(this.buildContextId())
                        .id(this.buildTaskId());
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
        return Task.builder()
                // 填充属性
                .artifacts(List.of(artifact.artifactId(String.valueOf(segment.getIndex()))
                        .metadata(segment.getMetadata())))
                .metadata(this.buildTaskMetadata(segment))
                .timestamp(this.buildTimestamp(segment))
                .status(this.buildTaskStatus(segment))
                .contextId(this.buildContextId())
                .id(this.buildTaskId())
                .build();
    }

    // 构建失败的Task
    protected Task buildFailedTask(Segment segment) throws Exception {
        return Task.builder()
                // Artifact没有Meta
                .artifacts(List.of(this.buildTextArtifact(segment)
                        .artifactId(String.valueOf(segment.getIndex()))
                        .metadata(segment.getMetadata()))
                )
                .metadata(this.buildTaskMetadata(segment))
                .timestamp(this.buildTimestamp(segment))
                .status(TaskStatus.builder()
                        // 强制指定状态
                        .state(TaskStatus.STATUS_FAILED)
                        .build())
                .contextId(this.buildContextId())
                .id(this.buildTaskId())
                .build();
    }

    // 构建失败的Task（Artifact ID = UUID）
    protected Task buildFailedTask(String content) throws Exception {
        return Task.builder()
                .timestamp(this.buildTimestamp(System.currentTimeMillis()))
                .artifacts(List.of(this.buildTextArtifact(content).artifactId(UUID.randomUUID().toString())))
                .status(TaskStatus.builder()
                        // 强制指定状态
                        .state(TaskStatus.STATUS_FAILED)
                        .build())
                .contextId(this.buildContextId())
                .id(this.buildTaskId())
                .build();
    }

    // 构建ContextID
    protected String buildContextId() throws Exception {
        return this.a2aRequest.getTrace();
    }

    // 构建TaskID
    protected String buildTaskId() throws Exception {
        return String.valueOf(this.a2aRequest.getId());
    }

    // 构建A2AID
    protected String buildA2AId() throws Exception {
        return String.valueOf(this.a2aRequest.getId());
    }

    // 写入网络
    protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
        this.a2aRequest.writeOnce(a2ACmdResponse);
    }

    // 流关闭
    protected void close(String content, String status) throws Exception {
        Task task = Task.builder()
                // 构建Text Part
                .artifacts(List.of(this.buildTextArtifact(content)))
                .contextId(this.buildContextId())
                .status(TaskStatus.builder()
                        // 强制指定状态
                        .state(status)
                        .build())
                .id(this.buildTaskId())
                .build();
        this.write(A2ACmdResponse.builder()
                .id(this.buildA2AId())
                .result(task.reset())
                .finished(true)
                .build());
    }

    protected void start() throws Exception {

    }
}