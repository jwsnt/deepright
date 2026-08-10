package ai.open.right.workflow.a2a.protocol;

import ai.open.right.workflow.a2a.A2AProtocol;
import lombok.*;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Setter
@Getter
@Builder
@AllArgsConstructor
@NoArgsConstructor
public class TaskArtifactUpdateEvent implements A2AProtocol {

    public static final String PROTOCOL = "@taskArtifactUpdateEvent";

    // 可选元数据
    protected Map<String, Object> metadata;

    protected Artifact artifact;

    // 服务器生成的唯一标识符（如UUID），用于在多个相关任务或交互中维护上下文
    protected String contextId;

    /**
     * If true, this is the final chunk of the artifact.
     */
    @Builder.Default
    protected Boolean lastChunk = false;

    @Builder.Default
    protected String internal = TaskArtifactUpdateEvent.PROTOCOL;

    /**
     * If true, the content of this artifact should be appended to a previously sent artifact with the same ID.
     */
    @Builder.Default
    protected Boolean append = false;

    // 对应Task的ID
    protected String taskId;

    @Builder.Default
    protected String kind = "artifact-update";

    public TaskArtifactUpdateEvent metadata(Map<String, Object> metadata) {
        if (this.metadata == null) {
            this.metadata = new HashMap<String,Object>();
        }
        for (String key : metadata.keySet()) {
            this.metadata.putIfAbsent(key, metadata.get(key));
        }
        return this;
    }

    public TaskArtifactUpdateEvent contextId(String contextId) {
        this.contextId = StringUtils.defaultIfBlank(this.contextId, contextId);
        return this;
    }

    public TaskArtifactUpdateEvent lastChunk(Boolean lastChunk) {
        this.lastChunk = this.lastChunk != null ? this.lastChunk : lastChunk;
        return this;
    }

    public TaskArtifactUpdateEvent append(Boolean append) {
        this.append = this.append != null ? this.append : append;
        return this;
    }

    public TaskArtifactUpdateEvent taskId(String id) {
        this.taskId = this.taskId != null ? this.taskId : id;
        return this;
    }

    @Override
    public TaskArtifactUpdateEvent reset() {
        this.internal = null;
        return this;
    }

    public Boolean isSupport(String internal) {
        return StringUtils.equalsIgnoreCase(this.internal, internal);
    }
}
