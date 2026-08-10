package ai.open.right.workflow.a2a.protocol;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;

@Setter
@Getter
@Builder
public class TaskStatusUpdateEvent {

    // 可选元数据
    protected Map<String, Object> metadata;

    // 任务的当前状态，包括其状态和描述性消息（TaskStatus）
    protected TaskStatus status;

    // 服务器生成的唯一标识符（如UUID），用于在多个相关任务或交互中维护上下文
    protected String contextId;

    /** If true, this is the final event in the stream for this interaction. */
    @Builder.Default
    @JsonProperty("final")
    protected Boolean finished = false;

    // 对应Task的ID
    protected String taskId;

    @Builder.Default
    protected String kind = "status-update";
}
