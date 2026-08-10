package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import lombok.*;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Builder
@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class ProviderFunCallRequest implements LLMFunCallRequest {

    @Builder.Default
    protected final Long created = System.currentTimeMillis();

    // 不同模型的额外参数
    protected Map<String, Object> metadata;

    protected Map<String, Object> refer;

    protected String reason;

    // 模型
    protected String model;

    protected Object args;

    protected String name;

    protected String api;

    protected String id;

    public Boolean isValid() {
        return !StringUtils.isEmpty(this.name);
    }

    public ProviderFunCallRequest setNameIfAbsent(String name) {
        this.name = StringUtils.defaultIfBlank(this.name, name);
        return this;
    }

    public ProviderFunCallRequest setIdIfAbsent(String id) {
        this.id = StringUtils.defaultIfBlank(this.id, id);
        return this;
    }

    public ProviderFunCallRequest appendArgs(Object args) {
        if (this.args == null) {
            this.args = args;
            return this;
        }
        if (args == null) {
            return this;
        }
        if (Map.class.isAssignableFrom(this.args.getClass())) {
            Map.class.cast(this.args).putAll(Map.class.cast(args));
        } else {
            this.args = this.args + String.valueOf(args);
        }
        return this;
    }

    @Override
    public void putMetadata(String key, Object value) {
        this.metadata = this.metadata != null ? this.metadata : new HashMap<String, Object>();
        this.metadata.put(key, value);
    }
}
