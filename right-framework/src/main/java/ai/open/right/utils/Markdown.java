package ai.open.right.utils;

import com.fasterxml.jackson.databind.JsonNode;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;

import java.util.Iterator;
import java.util.Map;

@Slf4j
public class Markdown {

    public static String array(String json) throws Exception {
        return Markdown.object(JsonUtils.write(MapUtils.getMap(JsonUtils.read(json, Map.class), "items")));
    }

    public static String object(String json) throws Exception {
        JsonNode root = JsonUtils.instance().readTree(json);
        JsonNode p = root.get("properties");
        JsonNode r = root.get("required");
        StringBuffer buffer = new StringBuffer();
        buffer.append("|Field|Type|Required|Description|").append(System.lineSeparator());
        buffer.append("|:---|:---|:---|:---|").append(System.lineSeparator());
        Iterator<String> f = p.fieldNames();
        while (f.hasNext()) {
            String name = f.next();
            JsonNode d = p.get(name);
            String t = "-";
            if (d.has("type")) {
                t = d.get("type").asText();
            }
            String description = "-";
            if (d.has("description")) {
                description = d.get("description").asText();
            }
            boolean isRequired = false;
            if (r != null && r.isArray()) {
                for (int i = 0; i < r.size(); i++) {
                    if (r.get(i).asText().equals(name)) {
                        isRequired = true;
                        break;
                    }
                }
            }
            buffer.append("|**").append(name).append("**|`").append(t).append("`|");
            buffer.append(isRequired ? Boolean.TRUE : Boolean.FALSE);
            buffer.append("|").append(description).append("|");
            buffer.append(System.lineSeparator());
        }
        return buffer.toString();
    }
}
