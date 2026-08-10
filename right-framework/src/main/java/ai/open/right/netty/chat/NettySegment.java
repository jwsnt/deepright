package ai.open.right.netty.chat;

import ai.open.right.netty.NettyStream;
import ai.open.right.workflow.flow.llm.LLMUsage;

import java.util.Map;

// 写入Netty的响应
public interface NettySegment extends NettyStream {

    public Map<String, Object> getMetadata();

    public LLMUsage getUsage();

    public String getWorkflow();

    public Long getTimestamp();

    public String getContent();

    public Boolean getStream();

    public Integer getIndex();

    public String getTrace();

    public Integer getCode();

    public String getBiz();

    public String getId();

    // 标记已读位置
    public void mark();
}
