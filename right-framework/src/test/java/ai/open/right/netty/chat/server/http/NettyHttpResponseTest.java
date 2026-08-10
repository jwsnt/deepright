package ai.open.right.netty.chat.server.http;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.chat.NettySegment;
import ai.open.right.workflow.flow.llm.LLMUsage;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

public class NettyHttpResponseTest {

    @Test
    public void testConstructorOnce() {
        NettySegment segment = EasyMock.createMock(NettySegment.class);
        LLMUsage usage = EasyMock.createMock(LLMUsage.class);
        Map<String, Object> meta = new HashMap<>();
        EasyMock.expect(segment.getId()).andReturn("sid").anyTimes();
        EasyMock.expect(segment.getTrace()).andReturn("trace-id").anyTimes();
        EasyMock.expect(segment.getCode()).andReturn(200).anyTimes();
        EasyMock.expect(segment.getUsage()).andReturn(usage).anyTimes();
        EasyMock.expect(segment.getTimestamp()).andReturn(123456789L).anyTimes();
        EasyMock.expect(segment.getWorkflow()).andReturn("workflow-id").anyTimes();
        EasyMock.expect(segment.getContent()).andReturn("hello").anyTimes();
        EasyMock.expect(segment.getMetadata()).andReturn(meta).anyTimes();
        EasyMock.expect(segment.getIndex()).andReturn(0).anyTimes();
        EasyMock.expect(segment.getBiz()).andReturn("biz").anyTimes();
        EasyMock.expect(segment.isFinished()).andReturn(true).anyTimes();

        EasyMock.replay(segment, usage);

        NettyHttpResponse response = new NettyHttpResponse(segment, false, false);

        Assertions.assertEquals("sid", response.getId());
        Assertions.assertEquals(200, response.getCode());
        Assertions.assertEquals(usage, response.getUsage());
        Assertions.assertEquals(123456789L, response.getCreated());
        Assertions.assertEquals("workflow-id", response.getWorkflow());
        Assertions.assertEquals(1, response.getChoices().size());
        Assertions.assertEquals("hello", response.getChoices().get(0).getMessage().getContent());
        Assertions.assertEquals("stop", response.getChoices().get(0).getFinishReason());

        EasyMock.verify(segment, usage);
    }

    @Test
    public void testConstructorStreamSse() {
        NettySegment segment = EasyMock.createMock(NettySegment.class);
        EasyMock.expect(segment.getId()).andReturn("sid").anyTimes();
        EasyMock.expect(segment.getTrace()).andReturn("trace-id").anyTimes();
        EasyMock.expect(segment.getCode()).andReturn(200).anyTimes();
        EasyMock.expect(segment.getUsage()).andReturn(null).anyTimes();
        EasyMock.expect(segment.getTimestamp()).andReturn(null).anyTimes();
        EasyMock.expect(segment.getWorkflow()).andReturn(null).anyTimes();
        EasyMock.expect(segment.getContent()).andReturn("hello").anyTimes();
        EasyMock.expect(segment.getMetadata()).andReturn(null).anyTimes();
        EasyMock.expect(segment.getIndex()).andReturn(null).anyTimes();
        EasyMock.expect(segment.isFinished()).andReturn(false).anyTimes();
        EasyMock.expect(segment.getBiz()).andReturn("biz").anyTimes();
        EasyMock.replay(segment);

        NettyHttpResponse response = new NettyHttpResponse(segment, true, true);

        Assertions.assertEquals("hello", response.getChoices().get(0).getDelta().getContent());
        Assertions.assertNull(response.getChoices().get(0).getFinishReason());
    }

    @Test
    public void testBuildReason() {
        NettyHttpResponse response = new NettyHttpResponse(ObjectBuilder.buildSegment(), false, false);
        SegmentDelegate s1 = SegmentDelegate.class.cast(ObjectBuilder.buildSegment());
        s1.setCode(200);
        s1.setFinished(true);
        SegmentDelegate s2 = SegmentDelegate.class.cast(ObjectBuilder.buildSegment());
        s2.setCode(200);
        s2.setFinished(false);
        SegmentDelegate s3 = SegmentDelegate.class.cast(ObjectBuilder.buildSegment());
        s3.setCode(500);
        Assertions.assertEquals("stop", response.buildReason(s1));
        Assertions.assertNull(response.buildReason(s2));
        Assertions.assertEquals("error", response.buildReason(s3));
    }
}
