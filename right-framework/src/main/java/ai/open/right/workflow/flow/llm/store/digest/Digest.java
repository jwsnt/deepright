package ai.open.right.workflow.flow.llm.store.digest;

import lombok.Getter;
import lombok.Setter;
import org.springframework.util.CollectionUtils;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Setter
@Getter
public class Digest {

    protected final Map<String, Object> digest;

    protected final List<String> keys;

    public Digest(Map<String, Object> digest, List<String> keys) {
        this.digest = digest;
        this.keys = keys;
    }

    public Digest merge(Map<String, Object> target) {
        if (!this.hasKeys()) {
            return this;
        }
        Map<String, Object> actual = new HashMap<String, Object>();
        for (String key : this.keys) {
            actual.put(key, this.digest.getOrDefault(key, target.get(key)));
        }
        this.digest.clear();
        this.digest.putAll(actual);
        return this;
    }

    public Boolean hasDigest() {
        return !CollectionUtils.isEmpty(this.digest);
    }

    public Boolean hasKeys() {
        return !CollectionUtils.isEmpty(this.keys);
    }
}
