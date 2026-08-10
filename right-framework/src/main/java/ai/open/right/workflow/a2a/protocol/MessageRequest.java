package ai.open.right.workflow.a2a.protocol;

import lombok.Getter;
import lombok.Setter;
import org.springframework.util.CollectionUtils;

import java.util.HashMap;
import java.util.Map;

// Message/Send和Message/Stream
@Setter
@Getter
public class MessageRequest {

    private Map<String, Object> metadata = new HashMap<String, Object>();

    protected Message message;

    // 合并Meta，已经存在的Key不覆盖
    public MessageRequest putIfAbsent(Map<String, ?> metadata) {
        if (CollectionUtils.isEmpty(metadata)) {
            return this;
        }
        for (String key : metadata.keySet()) {
            this.metadata.putIfAbsent(key, metadata.get(key));
        }
        return this;
    }
}
