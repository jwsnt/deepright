package ai.open.right.netty.chat.server.http;

import ai.open.right.netty.chat.NettySegment;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

import java.util.Map;
import java.util.UUID;

@Setter
@Getter
@Builder
// Netty解析转发过程中出现错误的Segment
public class NettyErrorSegment implements NettySegment {

    private static final NettyErrorUsage USAGE = new NettyErrorUsage();

    protected final String id = UUID.randomUUID().toString();

    protected String content;

    protected Integer code;

    @Override
    public Map<String, Object> getMetadata() {
        return null;
    }

    @Override
    public NettyErrorUsage getUsage() {
        return NettyErrorSegment.USAGE;
    }

    @Override
    public String getWorkflow() {
        return null;
    }

    @Override
    public Boolean getStream() {
        return null;
    }

    @Override
    public Long getTimestamp() {
        return null;
    }

    @Override
    public Integer getIndex() {
        return null;
    }

    @Override
    public String getTrace() {
        return null;
    }

    @Override
    public String getBiz() {
        return null;
    }

    @Override
    public String getId() {
        return this.id;
    }

    @Override
    public Boolean isFinished() {
        return true;
    }

    @Override
    public void mark() {

    }
}
