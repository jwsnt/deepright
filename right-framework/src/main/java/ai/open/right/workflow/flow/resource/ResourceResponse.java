package ai.open.right.workflow.flow.resource;

import lombok.Getter;
import lombok.Setter;

import java.util.HashMap;
import java.util.Map;

@Getter
@Setter
public class ResourceResponse {

    protected Map<String, String> headers;

    protected String content;

    public void addHeader(String key, String value) {
        this.headers = this.headers != null ? this.headers : new HashMap<String, String>();
        this.headers.put(key, value);
    }
}
