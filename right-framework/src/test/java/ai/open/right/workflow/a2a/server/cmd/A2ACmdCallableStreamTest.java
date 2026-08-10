package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.*;
import ai.open.right.workflow.flow.llm.Segment;
import com.fasterxml.jackson.core.JsonParseException;
import com.google.common.collect.ImmutableMap;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.net.SocketAddress;
import java.util.List;
import java.util.Map;

public class A2ACmdCallableStreamTest {

    @Test
    public void testCallSegment() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        MessageRequest messageRequest = new MessageRequest();
        final Segment segment = ObjectBuilder.buildSegment();
        A2ACmdCallableStream a2ACmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
            }
        };
        a2ACmdCallableStream.call(segment);
        Assert.assertNotNull(a2ACmdCallableStream.getMessageRequest());
        Assert.assertNotNull(a2ACmdCallableStream.getA2aRequest());
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testCallSegmentWithOut200() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2ACmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                TaskStatusUpdateEvent taskStatusUpdateEvent = TaskStatusUpdateEvent.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(taskStatusUpdateEvent.getStatus().getState(), TaskStatus.STATUS_FAILED);
                Assert.assertEquals(true, a2ACmdResponse.getFinished());
            }
        };
        Segment segment = ObjectBuilder.buildSegment(500);
        a2ACmdCallableStream.call(segment);
        Assert.assertNotNull(a2ACmdCallableStream.getMessageRequest());
        Assert.assertNotNull(a2ACmdCallableStream.getA2aRequest());
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testCallSegmentWithException() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2ACmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected A2ACmdResponse buildA2ACmdResponse(Segment segment, TaskArtifactUpdateEvent taskArtifactUpdateEvent) throws Exception {
                throw new RuntimeException();
            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                TaskStatusUpdateEvent taskStatusUpdateEvent = TaskStatusUpdateEvent.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(taskStatusUpdateEvent.getStatus().getState(), TaskStatus.STATUS_FAILED);
                Assert.assertEquals(true, a2ACmdResponse.getFinished());
            }
        };
        Segment segment = ObjectBuilder.buildSegment();
        a2ACmdCallableStream.call(segment);
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testBuildTextArtifact() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2aCmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                super.write(a2ACmdResponse);
                Assert.assertEquals(true, a2ACmdResponse.getFinished());
            }
        };
        Segment segment = ObjectBuilder.buildSegment();
        segment.setMetadata(ImmutableMap.of("A", "B"));
        segment.setContent("HELLO");
        TaskArtifactUpdateEvent taskArtifactUpdateEvent = a2aCmdCallableStream.buildTaskArtifactUpdateEvent(segment);
        Artifact artifact = taskArtifactUpdateEvent.getArtifact();
        Assert.assertNotNull(artifact.getArtifactId());
        Assert.assertTrue(artifact.getArtifactId().equals(String.valueOf(segment.getIndex())) || artifact.getArtifactId().length() == 36);
        Assert.assertEquals(segment.getMetadata(), artifact.getMetadata());
        Part part = artifact.getParts().getFirst();
        Assert.assertEquals(part.getKind(), Part.TEXT_KIND);
        Assert.assertEquals(segment.getContent(), part.getText());
        Assert.assertSame(taskArtifactUpdateEvent.getLastChunk(), segment.isFinished());
        Assert.assertNotSame(taskArtifactUpdateEvent.getAppend(), segment.isFinished());
        Assert.assertEquals(taskArtifactUpdateEvent.getTaskId(), a2aCmdCallableStream.buildTaskId());
    }

    @Test
    public void testBuildTextArtifactWithFinish() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2aCmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                super.write(a2ACmdResponse);
                Assert.assertEquals(false, a2ACmdResponse.getFinished());
            }
        };
        Segment segment = ObjectBuilder.buildSegmentWithOutFinish();
        segment.setMetadata(ImmutableMap.of("A", "B"));
        segment.setContent("HELLO");
        TaskArtifactUpdateEvent taskArtifactUpdateEvent = a2aCmdCallableStream.buildTaskArtifactUpdateEvent(segment);
        Artifact artifact = taskArtifactUpdateEvent.getArtifact();
        Assert.assertNotNull(artifact.getArtifactId());
        Assert.assertTrue(artifact.getArtifactId().equals(String.valueOf(segment.getIndex())) || artifact.getArtifactId().length() == 36);
        Assert.assertEquals(segment.getMetadata(), artifact.getMetadata());
        Part part = artifact.getParts().getFirst();
        Assert.assertEquals(part.getKind(), Part.TEXT_KIND);
        Assert.assertEquals(segment.getContent(), part.getText());
        Assert.assertSame(taskArtifactUpdateEvent.getLastChunk(), segment.isFinished());
        Assert.assertNotSame(taskArtifactUpdateEvent.getAppend(), segment.isFinished());
        Assert.assertEquals(taskArtifactUpdateEvent.getTaskId(), a2aCmdCallableStream.buildTaskId());
    }

    /**
     * 覆盖 A2ACmdCallableStream 88-93 行 JsonParseException catch 分支：无法解析 Json 时 log.debug(e.getMessage(), e) 与 buildTextArtifact(segment)。
     * 通过子类在 buildA2AData 中直接抛出 JsonParseException，保证进入 catch 并执行 buildTextArtifact(segment)。
     */
    @Test
    public void testBuildTaskArtifactUpdateEventJsonParseException() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(EasyMock.anyObject(NettyCloser.class))).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .context(context)
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2aCmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {
            @Override
            protected ai.open.right.workflow.a2a.A2AData buildA2AData(Segment segment) throws Exception {
                throw new JsonParseException(null, "invalid json");
            }
        };
        Segment segment = ObjectBuilder.buildSegment();
        segment.setContent("[invalid]");
        segment.setMetadata(ImmutableMap.of("K", "V"));
        TaskArtifactUpdateEvent event = a2aCmdCallableStream.buildTaskArtifactUpdateEvent(segment);
        Assert.assertNotNull(event.getArtifact());
        Assert.assertEquals(1, event.getArtifact().getParts().size());
        Assert.assertEquals("[invalid]", event.getArtifact().getParts().getFirst().getText());
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testSuccessTaskWithDetailTask() throws Exception {
        TaskArtifactUpdateEvent task = TaskArtifactUpdateEvent.builder()
                .taskId("TASKID")
                .lastChunk(false)
                .append(true)
                .contextId("CONTEXT")
                .metadata(ImmutableMap.of("A", "B"))
                .artifact(Artifact.builder()
                        .metadata(ImmutableMap.of("HELLO", "WORLD_1"))
                        .artifactId("MY ARTI")
                        .name("NAME")
                        .parts(List.of(Part.builder()
                                .text("HELLO")
                                .kind("TEXT")
                                .build()))
                        .build())
                .build();
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD_2"))
                .content(new StringBuffer(JsonUtils.write(task)))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .trace("TRACE")
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2ACmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                TaskArtifactUpdateEvent _task = TaskArtifactUpdateEvent.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getTaskId(), "TASKID");
                Assert.assertEquals(_task.getContextId(), "CONTEXT");
                Assert.assertEquals(_task.getMetadata().get("A"), "B");
                Assert.assertEquals(_task.getMetadata().get("HELLO"), "WORLD_2");
                Artifact artifact = _task.getArtifact();
                Assert.assertEquals("MY ARTI", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "WORLD_1");
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("HELLO", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
            }
        };
        a2ACmdCallableStream.call(segment);
    }

    @Test
    public void testSuccessTaskWithDetailArtifact() throws Exception {
        Artifact artifact = Artifact.builder()
                .metadata(ImmutableMap.of("HELLO_1", "WORLD"))
                .parts(List.of(Part.builder()
                        .kind("TEXT")
                        .text("DATA")
                        .build(), Part.builder()
                        .kind("DATA")
                        .data(ImmutableMap.of("A", "B"))
                        .build()))
                .artifactId("ARTID")
                .build();
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD_2"))
                .content(new StringBuffer(JsonUtils.write(artifact)))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .trace("TRACE")
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2ACmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                TaskArtifactUpdateEvent _task = TaskArtifactUpdateEvent.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getTaskId(), "req-007");
                Assert.assertEquals(_task.getContextId(), "TRACE");
                Assert.assertEquals(_task.getMetadata().get("HELLO"), "WORLD_2");
                Artifact artifact = _task.getArtifact();
                Assert.assertEquals("ARTID", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO_1"), "WORLD");
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "WORLD_2");
                Assert.assertEquals(2, artifact.getParts().size());
                Assert.assertEquals("DATA", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
                Assert.assertEquals("B", artifact.getParts().getLast().getData().get("A"));
                Assert.assertEquals("DATA", artifact.getParts().getLast().getKind());
            }
        };
        a2ACmdCallableStream.call(segment);
    }

    @Test
    public void testSuccessTaskWithSegment() throws Exception {
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD_2"))
                .content(new StringBuffer(JsonUtils.write(ImmutableMap.of("A","B"))))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .trace("TRACE")
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableStream a2ACmdCallableStream = new A2ACmdCallableStream(a2ARequest, messageRequest) {

            @Override
            protected void start() throws Exception {

            }

            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                TaskArtifactUpdateEvent _task = TaskArtifactUpdateEvent.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getTaskId(), "req-007");
                Assert.assertEquals(_task.getContextId(), "TRACE");
                Assert.assertEquals(_task.getMetadata().get("HELLO"), "WORLD_2");
                Artifact artifact = _task.getArtifact();
                Assert.assertEquals("99", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "WORLD_2");
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("B", artifact.getParts().getLast().getData().get("A"));
                Assert.assertEquals("data", artifact.getParts().getLast().getKind());
            }
        };
        a2ACmdCallableStream.call(segment);
    }
}
