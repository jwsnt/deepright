package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import lombok.*;
import org.springframework.util.StringUtils;

import java.util.HashMap;
import java.util.Map;

@Getter
@Setter
@Builder
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class ProviderFunCallResponse implements LLMFunCallResponse {

    @Builder.Default
    protected Long created = System.currentTimeMillis();

    protected Map<String,Object> metadata;

    protected String response;

    protected String model;

    protected String name;

    protected String id;

    @Override
    public void putMetadata(String key, Object value) {
        this.metadata = this.metadata != null ? this.metadata : new HashMap<String, Object>();
        this.metadata.put(key, value);
    }

    public Boolean isValid() {
        return StringUtils.hasText(this.name);
    }
}
