package ai.open.right.workflow.flow.media;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.StringUtils;

import java.nio.file.Files;
import java.nio.file.Paths;

@Setter
@Getter
@Slf4j
@ToString
public class MediaContext {

    public static final String PREFIX_INLINE = "inline:";

    public static final String TEXT = "text";

    protected String type;

    protected String data;

    public Boolean canEncodeBase64() {
        // Not Text Or inline:
        return StringUtils.hasText(this.type) && !(MediaContext.TEXT.equalsIgnoreCase(this.type) || MediaContext.isInline(this.type));
    }

    public String getType(String type) {
        return StringUtils.hasText(this.type) ? this.type : type;
    }

    public static Boolean isInline(String type) {
        return StringUtils.startsWithIgnoreCase(type, MediaContext.PREFIX_INLINE);
    }

    public static String pureType(String type) {
        return StringUtils.hasText(type) ? type.replaceFirst(MediaContext.PREFIX_INLINE, "") : type;
    }

    public static String mimeType(String file) {
        try {
            // 预测MimeType
            return Files.probeContentType(Paths.get(file));
        } catch (Exception e) {
            log.warn(e.getMessage(), e);
            return "";
        }
    }
}
