package ai.open.right.workflow.flow.resource;

import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Getter
@Setter
public class ResourceRequest {

    public static final String METHOD_GET = "GET";

    protected Map<String, String> headers;

    protected Map<String, Object> content;

    protected String method = ResourceRequest.METHOD_GET;

    protected String url;

    public Boolean isValid() {
        return !StringUtils.isEmpty(this.url) && !StringUtils.isEmpty(this.method);
    }

    public void putContent(String key, Object value) {
        if (this.content == null) {
            this.content = new HashMap<String, Object>();
        }
        this.content.put(key, value);
    }

    public void putHeader(String key, String value) {
        if (this.headers == null) {
            this.headers = new HashMap<String, String>();
        }
        this.headers.put(key, value);
    }

    public Boolean hasHeaders() {
        return !MapUtils.isEmpty(this.headers);
    }
}
