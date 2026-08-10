package ai.open.right.workflow.mcp.client.utils;

import org.springframework.util.CollectionUtils;

import java.util.*;
import java.util.stream.Collectors;

public class McpToolsUtils {

    // 允许的顶级字段
    private static final Set<String> ALLOWED_TOTAL_FIELDS = Set.of(
            "type", "nullable", "required", "format", "description",
            "properties", "items", "enum", "anyOf"
    );

    // anyOf数组中每个对象允许的字段
    private static final Set<String> ALLOWED_ANY_OF_FIELDS = Set.of("type", "description");

    public static Map<String, Object> filter(Map<String, Object> properties) {
        if (CollectionUtils.isEmpty(properties)) {
            return properties;
        }
        Map<String, Object> filter = new HashMap<String, Object>();
        for (String key : properties.keySet()) {
            Object val = properties.get(key);
            if (Map.class.isAssignableFrom(val.getClass())) {
                Map<String, Object> value = Map.class.cast(properties.get(key));
                value = value.entrySet().stream()
                        .filter(entry -> ALLOWED_TOTAL_FIELDS.contains(entry.getKey()))
                        .collect(Collectors.toMap(Map.Entry::getKey, McpToolsUtils::filterValue));
                filter.put(key, value);
            } else {
                filter.put(key, val);
            }
        }
        return filter;
    }

    @SuppressWarnings("unchecked")
    private static Object filterValue(Map.Entry<String, Object> entry) {
        String key = entry.getKey();
        Object value = entry.getValue();
        // 处理anyOf字段 - 它的值是一个列表
        if ("anyOf".equals(key) && value instanceof Iterable<?>) {
            List<Object> filteredList = new ArrayList<>();
            ((Iterable<?>) value).forEach(item -> {
                // 过滤anyOf中的每个对象，只保留允许的字段
                Map<String, Object> filteredItem = ((Map<String, Object>) item).entrySet().stream()
                        .filter(e -> ALLOWED_ANY_OF_FIELDS.contains(e.getKey()))
                        .collect(Collectors.toMap(Map.Entry::getKey, e -> McpToolsUtils.filterValue(e)));
                filteredList.add(filteredItem);
            });
            return filteredList;
        }
        // 处理items字段 - 递归过滤
        if ("items".equals(key) && value instanceof Map<?, ?>) {
            return filter((Map<String, Object>) value);
        }
        // 处理enum字段 - 它是一个列表，不需要过滤其内容
        if ("enum".equals(key) && value instanceof Iterable<?>) {
            return value; // enum的值直接保留
        }
        // 其他类型的值直接保留
        return value;
    }
}
    