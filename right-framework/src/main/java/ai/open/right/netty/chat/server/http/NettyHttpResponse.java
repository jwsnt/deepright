package ai.open.right.netty.chat.server.http;

import ai.open.right.netty.chat.NettySegment;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.LLMUsage;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@Setter
@Getter
// Open AI Response
public class NettyHttpResponse {

    private List<Choice> choices = new ArrayList<Choice>();

    protected LLMUsage usage;

    protected String workflow;

    protected String object = "chat.completion";

    protected String model = "right";

    protected Long created;

    // 报文状态码
    protected Integer code;

    protected String biz;

    protected String id;

    public NettyHttpResponse(NettySegment nettySegment, Boolean stream, Boolean sse) {
        this.setId(nettySegment.getId());
        this.setBiz(nettySegment.getBiz());
        this.setUsage(nettySegment.getUsage());
        this.setCreated(nettySegment.getTimestamp());
        this.setWorkflow(nettySegment.getWorkflow());
        this.setCode(ProtocolCode.mapping(nettySegment.getCode()));
        NettyHttpResponse.Choice choice = new NettyHttpResponse.Choice();
        // 如果为SSE响应则不标记Finish_Reason，否则根据Segment构建
        choice.setFinishReason(sse ? null : this.buildReason(nettySegment));
        NettyHttpMessage httpMessage = new NettyHttpMessage();
        httpMessage.setContent(nettySegment.getContent());
        choice.setMetadata(nettySegment.getMetadata());
        choice.setIndex(nettySegment.getIndex());
        // 响应类型
        if (!stream) {
            choice.setMessage(httpMessage);
        } else {
            choice.setDelta(httpMessage);
        }
        this.choices.add(choice);
    }

    public String buildReason(NettySegment nettySegment) {
        // 2xx Code时已完成标记Stop，非200 Code时标记Error
        return ProtocolCode.range2xx(nettySegment.getCode()) ? (nettySegment.isFinished() ? "stop" : null) : "error";
    }

    @Setter
    @Getter
    public static class Choice {

        @JsonProperty("metadata")
        protected Map<String, Object> metadata;

        @JsonProperty("message")
        protected NettyHttpMessage message;

        @JsonProperty("delta")
        protected NettyHttpMessage delta;

        @JsonProperty("finish_reason")
        protected String finishReason;

        protected Integer index = 0;
    }
}