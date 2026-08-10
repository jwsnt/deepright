package ai.open.right.workflow.flow.llm.provider.coze;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderStream;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

import java.util.List;
import java.util.Map;

@Slf4j
public class CozeStream extends ProviderStream<CozeRequest> {

    public CozeStream(ProviderStreamConfig<CozeRequest> providerRequestConfig) throws Exception {
        super(providerRequestConfig);
    }

    @Override
    protected Boolean stream(String source) throws Exception {
        Assert.isTrue(StringUtils.startsWithIgnoreCase(source, "data:"), "Invalid data");
        Map<String, Object> body = JsonUtils.read(source.replaceFirst("data:", ""), Map.class);
        // {"event":"done"}
        Object event = body.get("event");
        if (event == null || "done".equalsIgnoreCase(event.toString())) {
            return true;
        }
        Boolean finished = Boolean.class.cast(body.get("is_finish"));
        Integer seqId = Integer.class.cast(body.get("seq_id"));
        Map<String, String> message = Map.class.cast(body.get("message"));
        Assert.notEmpty(message, "Message can not be empty");
        String content = message.getOrDefault("content", "");
        this.addContent(content, false);
        this.notify(seqId, finished);
        if (finished) {
            // 记忆
            this.storeConversation();
        }
        return finished;
    }

    @Override
    protected Boolean atonce(String source) throws Exception {
        Map<String, Object> body = JsonUtils.read(source, Map.class);
        List<Map<String, String>> messages = List.class.cast(body.get("messages"));
        Assert.notEmpty(messages, "Message can not be empty");
        for (Map<String, String> each : messages) {
            String type = each.get("type");
            String role = each.get("role");
            if ("assistant".equals(role) && "answer".equals(type)) {
                String text = each.get("content");
                if (StringUtils.hasText(text)) {
                    this.addContent(text, false);
                }
                break;
            }
        }
        this.notifySegment();
        return true;
    }

    @Override
    protected void tokenStatistic(Map<String, Object> body) throws Exception {

    }
}
