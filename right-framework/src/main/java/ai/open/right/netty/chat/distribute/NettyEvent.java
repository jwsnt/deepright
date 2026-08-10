package ai.open.right.netty.chat.distribute;

import ai.open.right.listener.Event;
import com.fasterxml.jackson.annotation.JsonIgnore;

import java.util.HashMap;
import java.util.Map;

public class NettyEvent implements Event {

    public static final String TYPE = "netty";

    @JsonIgnore
    protected final NettyRequest request;

    public NettyEvent(NettyRequest request) {
        this.request = request;
    }

    @Override
    public String getType() {
        return NettyEvent.TYPE;
    }

    @Override
    public Object getData() {
        // 提取关键数据
        Map<String, Object> body = new HashMap<String, Object>();
        body.put("mediaContext", this.request.getMediaContext());
        body.put("conversation", this.request.getConversation());
        body.put("histories", this.request.getHistories());
        body.put("metadata", this.request.getMetadata());
        body.put("query", this.request.getQuery());
        return body;
    }

    @Override
    public NettyEvent init() {
        return this;
    }

    @Override
    public Long getNow() {
        return this.request.getCreated();
    }

    @Override
    public String getBiz() {
        return this.request.getBiz();
    }

    @Override
    public String getChat() {
        return this.request.getChat();
    }

    @Override
    public String getDevice() {
        return this.request.getDevice();
    }

    @Override
    public String getWorkflow() {
        return this.request.getWorkflow();
    }

    @Override
    public String getDimension() {
        return this.request.getDimension();
    }
}
