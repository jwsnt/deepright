package ai.open.right.workflow.a2a.protocol;

import ai.open.right.workflow.a2a.A2AProtocol;
import lombok.*;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

// 任务期间生成的文件、数据结构或其他资源
@Setter
@Getter
@Builder
@AllArgsConstructor
@NoArgsConstructor
public class Artifact implements A2AProtocol {

    public static final String PROTOCOL = "@artifact";

    // 可选元数据
    protected Map<String, Object> metadata;

    // 可读描述
    protected String description;

    // 任务范围内工件的唯一标识符（如UUID），如Segment的Index
    protected String artifactId;

    @Builder.Default
    protected String internal = Artifact.PROTOCOL;

    // 内容数组
    protected List<Part> parts;

    // 可读名称
    protected String name;

    public Artifact metadata(Map<String, Object> metadata) {
        if (this.metadata == null) {
            this.metadata = new HashMap<String, Object>();
        }
        for (String key : metadata.keySet()) {
            this.metadata.putIfAbsent(key, metadata.get(key));
        }
        return this;
    }

    public Artifact artifactId(String artifactId) {
        this.artifactId = StringUtils.defaultIfBlank(this.artifactId, artifactId);
        return this;
    }

    @Override
    public Artifact reset() {
        this.internal = null;
        return this;
    }

    public Boolean isSupport(String internal) {
        return StringUtils.equalsIgnoreCase(this.internal, internal);
    }
}
    