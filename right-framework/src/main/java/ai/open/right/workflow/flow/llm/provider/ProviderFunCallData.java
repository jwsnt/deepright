package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.LLMFunCallData;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.collections.MapUtils;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Getter
@Setter
@ToString
// Fun Call的响应
public class ProviderFunCallData implements LLMFunCallData {

    protected List<LLMFunCallResponse> responses = new ArrayList<LLMFunCallResponse>();

    protected List<LLMFunCallRequest> requests = new ArrayList<LLMFunCallRequest>();

    protected Map<String, Object> metadata;

    public void addFunCall(LLMFunCallRequest funCallRequest, LLMFunCallResponse funCallResponse) {
        Assert.isTrue(funCallResponse.isValid(), "Fun call response is invalid");
        Assert.isTrue(funCallRequest.isValid(), "Fun call request is invalid");
        this.responses.add(funCallResponse);
        this.requests.add(funCallRequest);
    }

    public Boolean isValid() {
        return !CollectionUtils.isEmpty(this.responses) && !CollectionUtils.isEmpty(this.requests);
    }

    @Override
    public Map<String, Object> getMetadata() {
        this.metadata = this.metadata != null ? this.metadata : new HashMap<String, Object>();
        return this.metadata;
    }

    @Override
    public <T> T getMetadata(String key, Class<T> clazz) throws Exception {
        Assert.notNull(clazz, "The class can not be null");
        if (!MapUtils.isEmpty(this.metadata)) {
            Object val = this.metadata.get(key);
            if (val != null) {
                return clazz.isAssignableFrom(val.getClass()) ? clazz.cast(val) : JsonUtils.transfer(val, clazz);
            } else {
                return null;
            }
        } else {
            return null;
        }
    }

    @Override
    public void putMetadata(String key, Object value) {
        this.getMetadata().put(key, value);
    }
}
