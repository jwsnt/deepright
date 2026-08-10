package ai.open.right.workflow.a2a.protocol;

import ai.open.right.utils.JsonUtils;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.beanutils.PropertyUtils;

import java.util.Map;

@Getter
@Setter
public class DataPart {

    protected Map<String, Object> data;

    public DataPart(Map<String, Object> data) {
        this.data = data;
    }

    public <T> T getObject(String key, Class<T> clazz) throws Exception {
        Object value = this.getObject(key);
        if (value == null || value.getClass().isAssignableFrom(clazz)) {
            return clazz.cast(value);
        }
        return JsonUtils.transfer(value, clazz);
    }

    public Object getObject(String key) throws Exception {
        return PropertyUtils.getNestedProperty(this.data, key);
    }

    public Boolean equals(String key, Object source) throws Exception {
        Object target = this.getObject(key);
        return source.equals(target);
    }
}
