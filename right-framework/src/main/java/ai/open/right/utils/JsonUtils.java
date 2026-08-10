package ai.open.right.utils;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.core.JsonFactory;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.StreamReadConstraints;
import com.fasterxml.jackson.core.StreamReadFeature;
import com.fasterxml.jackson.core.json.JsonReadFeature;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.MapperFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;

import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Slf4j
public class JsonUtils {

    // 操作系统兼容
    public static final Boolean TRANSFER = System.getProperty("os.name").toLowerCase().contains("win");

    public static final String PREFIX_JAVASCRIPT = "```javascript";

    public static final String PATTERN = "```json\\s*([\\s\\S]*?)```";

    public static final String PREFIX_JSON = "```json";

    public static final String SUFFIX = "```";

    public static ObjectMapper MAPPER;

    public static JsonFactory FACTORY;

    static {
        JsonUtils.FACTORY = new JsonFactory();
        // 由使用者保证大小不产生DDOS（上层Netty有MaxContentLength限制）
        String maxValue = System.getProperty("json.maxStringLength");
        JsonUtils.FACTORY.setStreamReadConstraints(StreamReadConstraints.builder().maxStringLength(!StringUtils.isEmpty(maxValue) ? Integer.parseInt(maxValue) : Integer.MAX_VALUE).build());
        JsonUtils.MAPPER = new ObjectMapper(JsonUtils.FACTORY);
        // 开启容错开关：允许 JSON 字符串中出现未转义的控制字符（如换行符、制表符等）
        JsonUtils.MAPPER.configure(JsonReadFeature.ALLOW_UNESCAPED_CONTROL_CHARS.mappedFeature(), true);
        JsonUtils.MAPPER.configure(StreamReadFeature.INCLUDE_SOURCE_IN_LOCATION.mappedFeature(), true);
        JsonUtils.MAPPER.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        JsonUtils.MAPPER.configure(SerializationFeature.WRITE_NULL_MAP_VALUES, false);
        JsonUtils.MAPPER.configure(JsonParser.Feature.ALLOW_SINGLE_QUOTES, true);
        // 宽松转换：比如字符串数字自动转数字类型
        JsonUtils.MAPPER.enable(DeserializationFeature.ACCEPT_EMPTY_STRING_AS_NULL_OBJECT);
        JsonUtils.MAPPER.enable(DeserializationFeature.ACCEPT_SINGLE_VALUE_AS_ARRAY);
        JsonUtils.MAPPER.enable((DeserializationFeature.UNWRAP_SINGLE_VALUE_ARRAYS));
        // 忽略大小写
        JsonUtils.MAPPER.enable(MapperFeature.ACCEPT_CASE_INSENSITIVE_PROPERTIES);
        JsonUtils.MAPPER.setSerializationInclusion(JsonInclude.Include.NON_NULL);
    }

    public static ObjectMapper instance() {
        return JsonUtils.MAPPER;
    }

    public static <T> T read(InputStream t, Class<T> c) throws Exception {
        if (t != null) {
            try (InputStream input = t) {
                return c.equals(String.class) ? c.cast(IOUtils.toString(input, StandardCharsets.UTF_8)) : JsonUtils.MAPPER.readValue(t, c);
            }
        } else {
            return null;
        }
    }

    public static <T> T read(byte[] t, Class<T> c) throws Exception {
        if (t != null) {
            return c.equals(String.class) ? c.cast(new String(t, StandardCharsets.UTF_8)) : JsonUtils.MAPPER.readValue(t, c);
        } else {
            return null;
        }
    }

    public static <T> T read(String t, Class<T> c) throws Exception {
        if (t != null) {
            return c.equals(String.class) ? c.cast(t) : JsonUtils.MAPPER.readValue(JsonUtils.clean(t), c);
        } else {
            return null;
        }
    }

    // OutputStream由外部控制关闭
    public static void write(OutputStream output, Object t) throws Exception {
        if (output != null) {
            if (String.class.isAssignableFrom(t.getClass())) {
                output.write(t.toString().getBytes(StandardCharsets.UTF_8));
            } else {
                JsonUtils.MAPPER.writeValue(output, t);
            }
        }
    }

    public static String write(Object t) throws Exception {
        return t == null ? null : (String.class.isAssignableFrom(t.getClass()) ? t.toString() : JsonUtils.MAPPER.writeValueAsString(t));
    }

    // 对象转换
    public static <T> T transfer(Object t, Class<T> c) throws Exception {
        if (t != null) {
            String val = JsonUtils.write(t);
            return JsonUtils.read(val, c);
        } else {
            return null;
        }
    }

    // 提取JSON
    public static String extract(String script) throws Exception {
        if (!StringUtils.isEmpty(script)) {
            Matcher matcher = Pattern.compile(JsonUtils.PATTERN).matcher(script);
            if (matcher.find()) {
                String group = matcher.group(1).trim();
                // 用于Window特殊转义
                return JsonUtils.clean(JsonUtils.TRANSFER ? group.replaceAll("\\\"", "\\\\\\\"") : group);
            }
            return script;
        } else {
            return script;
        }
    }

    // 清理大模型产生的json/js前后缀
    public static String clean(String t) throws Exception {
        if (!StringUtils.isEmpty(t)) {
            String s = t.trim();
            // 如果字符串正好是前缀或后缀，返回空字符串，防止越界并处理不完整块
            if (s.equals(JsonUtils.SUFFIX) || s.equals(JsonUtils.PREFIX_JSON) || s.equals(JsonUtils.PREFIX_JAVASCRIPT)) {
                return "";
            }
            // 增加长度检查，防止 substring/delete 越界 (例如输入仅为 ``` 时)
            if (StringUtils.startsWithIgnoreCase(s, JsonUtils.PREFIX_JAVASCRIPT) && s.endsWith(JsonUtils.SUFFIX) && s.length() >= JsonUtils.PREFIX_JAVASCRIPT.length() + JsonUtils.SUFFIX.length()) {
                s = s.substring(JsonUtils.PREFIX_JAVASCRIPT.length(), s.length() - JsonUtils.SUFFIX.length());
            } else if (StringUtils.startsWithIgnoreCase(s, JsonUtils.PREFIX_JSON) && s.endsWith(JsonUtils.SUFFIX) && s.length() >= JsonUtils.PREFIX_JSON.length() + JsonUtils.SUFFIX.length()) {
                s = s.substring(JsonUtils.PREFIX_JSON.length(), s.length() - JsonUtils.SUFFIX.length());
            } else if (StringUtils.startsWithIgnoreCase(s, JsonUtils.SUFFIX) && s.endsWith(JsonUtils.SUFFIX) && s.length() >= JsonUtils.SUFFIX.length() * 2) {
                s = s.substring(JsonUtils.SUFFIX.length(), s.length() - JsonUtils.SUFFIX.length());
            }
            s = s.trim();
            if (log.isDebugEnabled()) {
                log.debug("Clean json={}", s);
            }
            // 兜底，如果已经是JSON则返回，否则尝试抽取
            return (s.startsWith("[") && s.endsWith("]")) || (s.startsWith("{") && s.endsWith("}")) ? s : JsonUtils.extract(s);
        } else {
            return t;
        }
    }

    // 是否可能JSON
    public static Boolean like(String t) throws Exception {
        if (t != null) {
            String json = JsonUtils.clean(t);
            return (json.startsWith("[") && json.endsWith("]")) || (json.startsWith("{") && json.endsWith("}"));
        } else {
            return false;
        }
    }

    public static Boolean array(String t) throws Exception {
        if (t != null) {
            String json = JsonUtils.clean(t);
            return (json.startsWith("[") && json.endsWith("]"));
        } else {
            return false;
        }
    }

    public static Boolean map(String t) throws Exception {
        if (t != null) {
            String json = JsonUtils.clean(t);
            return (json.startsWith("{") && json.endsWith("}"));
        } else {
            return false;
        }
    }
}

