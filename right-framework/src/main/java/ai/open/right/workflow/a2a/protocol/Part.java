package ai.open.right.workflow.a2a.protocol;

import lombok.*;
import org.apache.commons.lang3.StringUtils;

import java.util.Map;

// 定义所有消息或工件部分共有的基本属性
@Setter
@Getter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Part {

    public final static String DATA_KIND = "data";

    public final static String FILE_KIND = "file";

    public final static String TEXT_KIND = "text";

    // 与此部分相关联的可选元数据
    protected Map<String, Object> metadata;

    // 结构化数据内容
    protected Map<String, Object> data;

    protected FileData file;

    // 文本部分的字符串内容
    protected String text;

    protected String kind = Part.TEXT_KIND;

    public Boolean isKind(String kind) {
        return StringUtils.equalsIgnoreCase(this.kind, kind);
    }
}
    