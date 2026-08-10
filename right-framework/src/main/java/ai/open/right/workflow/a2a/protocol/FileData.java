package ai.open.right.workflow.a2a.protocol;

import lombok.*;
import org.apache.commons.lang3.StringUtils;

// 表示内容直接作为base64编码字符串提供的文件
@Setter
@Getter
@Builder
@AllArgsConstructor
@NoArgsConstructor
public class FileData {

    /**
     * The MIME type of the file (e.g., "application/pdf").
     */
    protected String mimeType;

    // 与uri互斥
    protected String bytes;

    protected String name;

    // 与bytes互斥
    protected String uri;

    public String getContent() {
        return StringUtils.defaultString(this.bytes, this.uri);
    }

    public Boolean isBytes() {
        return StringUtils.isEmpty(this.bytes);
    }

    public Boolean isUri() {
        return StringUtils.isEmpty(this.uri);
    }
}
    