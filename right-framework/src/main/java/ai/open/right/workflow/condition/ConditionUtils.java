package ai.open.right.workflow.condition;

import ai.open.right.utils.JsonUtils;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.BooleanUtils;

import java.util.Map;

@Slf4j
public class ConditionUtils {

    public static final String KEY = "condition";

    public static final String VAL = "content";

    public static Condition checkCondition(Object content) throws Exception {
        if (log.isDebugEnabled()) {
            log.debug("Condition content={}", content);
        }
        if (content == null) {
            return Condition.FALSE;
        }
        String content2string = content.toString();
        if (JsonUtils.like(content2string)) {
            Map<String, Object> data = JsonUtils.read(content2string, Map.class);
            Object condition = MapUtils.getObject(data, ConditionUtils.KEY);
            if (condition != null) {
                return Condition.builder()
                        .condition(BooleanUtils.toBoolean(condition.toString()))
                        .content(MapUtils.getString(data, ConditionUtils.VAL))
                        .build();
            }
        }
        return Condition.builder()
                .condition(BooleanUtils.toBoolean(content2string))
                .build();
    }
}
