package ai.open.right.workflow.mcp.client.utils;

import lombok.extern.slf4j.Slf4j;
import org.springframework.util.Assert;

import java.util.Map;

@Slf4j
// 解析内容
public class McpContentUtils {

    public static String resource(String type, Map<String, Object> content) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Mcp resource={}-{}", type, content);
        }
        if (McpContentUtils.isResource(type)) {
            Map<String, Object> resource = Map.class.cast(content.get("resource"));
            String mimeType = String.class.cast(resource.get("mimeType"));
            Assert.hasText(mimeType, "Mime type can not be empty");
            if (McpContentUtils.isTextPlain(mimeType)) {
                return McpContentUtils.doTextPlan(resource);
            } else {
                return McpContentUtils.doText(resource);
            }
        }
        if (McpContentUtils.isTextPlain(type)) {
            return McpContentUtils.doTextPlan(content);
        }
        if (McpContentUtils.isText(type)) {
            return McpContentUtils.doText(content);
        }
        if (log.isWarnEnabled()) {
            log.warn("Mcp resource will be empty");
        }
        return null;
    }

    public static String error(Map<String, Object> content) throws Exception {
        String type = String.class.cast(content.get("type"));
        if (McpContentUtils.isTextPlain(type) || McpContentUtils.isText(type)) {
            return McpContentUtils.doText(content);
        }
        return null;
    }

    public static String doTextPlan(Map<String, Object> content) {
        return String.class.cast(content.get("text"));
    }

    public static String doText(Map<String, Object> content) {
        return String.class.cast(content.get("text"));
    }

    public static Boolean isTextPlain(String type) {
        return "text/plain".equalsIgnoreCase(type);
    }

    public static Boolean isResource(String type) {
        return "resource".equalsIgnoreCase(type);
    }

    public static Boolean isText(String type) {
        return "text".equalsIgnoreCase(type);
    }
}
