package ai.open.right.utils;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.dataformat.yaml.YAMLGenerator;
import com.fasterxml.jackson.dataformat.yaml.YAMLMapper;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import java.io.InputStream;
import java.util.Map;

@Slf4j
public class YamlUtils {

    public static YAMLMapper MAPPER;

    static {
        YamlUtils.MAPPER = new YAMLMapper(new YAMLFactory()
                // 去掉 ---
                .disable(YAMLGenerator.Feature.WRITE_DOC_START_MARKER));
        // 1. 去掉 null
        YamlUtils.MAPPER.setSerializationInclusion(JsonInclude.Include.NON_NULL);
        // 2. Map 排序
        YamlUtils.MAPPER.enable(SerializationFeature.ORDER_MAP_ENTRIES_BY_KEYS);
        // 3. 输出缩进
        YamlUtils.MAPPER.enable(SerializationFeature.INDENT_OUTPUT);
    }

    public static YAMLMapper instance() {
        return YamlUtils.MAPPER;
    }

    public static Map<String, Object> read(InputStream t) throws Exception {
        if (t != null) {
            try (InputStream input = t) {
                return YamlUtils.MAPPER.readValue(input, Map.class);
            }
        } else {
            return null;
        }
    }

    public static Map<String, Object> read(String t) throws Exception {
        if (t != null) {
            return YamlUtils.MAPPER.readValue(StringUtils.trim(t), Map.class);
        } else {
            return null;
        }
    }

    public static String write(Object t) throws Exception {
        return t == null ? null : (String.class.isAssignableFrom(t.getClass()) ? t.toString() : YamlUtils.MAPPER.writeValueAsString(t));
    }
}
