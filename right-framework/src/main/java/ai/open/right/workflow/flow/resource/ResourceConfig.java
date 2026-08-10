package ai.open.right.workflow.flow.resource;

import ai.open.right.workflow.flow.config.GlobalConfig;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;

import java.util.Map;

@Getter
@Setter
public class ResourceConfig extends GlobalConfig {

    protected Map<String, String> headers;

    protected Boolean autoCopy;

    protected Integer timeout;

    public ResourceConfig merge(ResourceConfig resourceConfig) throws Exception {
        super.merge(resourceConfig);
        if (resourceConfig != null) {
            this.timeout = this.timeout != null ? this.timeout : resourceConfig.timeout;
        }
        return this;
    }

    public Boolean getAutoCopy() {
        return this.autoCopy != null ? this.autoCopy : false;
    }

    public Boolean hasHeaders() {
        return !MapUtils.isEmpty(this.headers);
    }
}
