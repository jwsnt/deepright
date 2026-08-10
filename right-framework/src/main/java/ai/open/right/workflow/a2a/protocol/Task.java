package ai.open.right.workflow.a2a.protocol;

import ai.open.right.workflow.a2a.A2AProtocol;
import lombok.*;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

// 表示客户端和代理之间的单个有状态操作或对话
@Setter
@Getter
@Builder
@ToString
@AllArgsConstructor
@NoArgsConstructor
public class Task implements A2AProtocol {

    public static final String PROTOCOL = "@task";

    // 可选元数据
    protected Map<String, Object> metadata;

    // 代理在任务执行期间生成的工件集合
    protected List<Artifact> artifacts;

    // 任务期间交换的消息数组，代表对话历史
    protected List<Message> history;

    // 任务的当前状态，包括其状态和描述性消息（TaskStatus）
    protected TaskStatus status;

    // 服务器生成的唯一标识符（如UUID），用于在多个相关任务或交互中维护上下文
    protected String contextId;

    // 当前时间戳
    protected String timestamp;

    // 内部标记
    @Builder.Default
    protected String internal = Task.PROTOCOL;

    @Builder.Default
    protected String kind = "task";

    // 任务的唯一标识符（如UUID），由服务器为新任务生成
    protected Object id;

    public Task metadata(Map<String, Object> metadata) {
        if (this.metadata == null) {
            this.metadata = new HashMap<String, Object>();
        }
        for (String key : metadata.keySet()) {
            this.metadata.putIfAbsent(key, metadata.get(key));
        }
        return this;
    }

    public Task contextId(String contextId) {
        this.contextId = StringUtils.defaultIfBlank(this.contextId, contextId);
        return this;
    }

    public Task timestamp(String timestamp) {
        this.timestamp = StringUtils.defaultIfBlank(this.timestamp, timestamp);
        return this;
    }

    public Task status(TaskStatus status) {
        this.status = this.status != null ? this.status : status;
        return this;
    }

    public Task id(String id) {
        this.id = this.id != null ? this.id : id;
        return this;
    }

    public Task reset() {
        // 清除标记
        this.internal = null;
        return this;
    }

    public Boolean isSupport(String internal) {
        return StringUtils.equalsIgnoreCase(this.internal, internal);
    }
}
    