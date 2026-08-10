package ai.open.right.workflow.mcp.client;

import ai.open.right.utils.JsonUtils;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.beanutils.PropertyUtils;

import java.util.HashMap;
import java.util.Map;

@Getter
@Setter
public class McpData {

    protected Map<String, Object> data;

    public McpData(Map<String, Object> data) throws Exception {
        this.data = data;
    }

    public McpData() throws Exception {
        this.data = new HashMap<String, Object>();
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
