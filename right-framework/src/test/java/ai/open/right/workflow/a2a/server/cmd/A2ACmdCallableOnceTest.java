package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.ObjectBuilder;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2AData;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.*;
import ai.open.right.workflow.flow.llm.Segment;
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
import java.util.Collections;
import java.util.List;
import java.util.Map;

public class A2ACmdCallableOnceTest {

    @Test
    public void testCallSegment() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
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
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(TaskStatus.STATUS_COMPLETED, Task.class.cast(a2ACmdResponse.getResult()).getStatus().getState());
                Assert.assertEquals(true, a2ACmdResponse.getFinished());
            }
        };
        Segment segment = ObjectBuilder.buildSegment();
        a2ACmdCallableOnce.call(segment);
        Assert.assertNotNull(a2ACmdCallableOnce.getMessageRequest());
        Assert.assertNotNull(a2ACmdCallableOnce.getA2aRequest());
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testCallSegmentWithNot200() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn(NettyAttributes.HTTP_CORS).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
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
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(TaskStatus.STATUS_FAILED, Task.class.cast(a2ACmdResponse.getResult()).getStatus().getState());
                Assert.assertEquals(true, a2ACmdResponse.getFinished());
            }
        };
        Segment segment = ObjectBuilder.buildSegment(500);
        a2ACmdCallableOnce.call(segment);
        Assert.assertNotNull(a2ACmdCallableOnce.getMessageRequest());
        Assert.assertNotNull(a2ACmdCallableOnce.getA2aRequest());
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
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
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
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected A2ACmdResponse buildA2ACmdResponse(Segment segment, Task task) throws Exception {
                throw new RuntimeException();
            }
        };
        Segment segment = ObjectBuilder.buildSegment();
        a2ACmdCallableOnce.call(segment);
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testBuildTextArtifact() throws Exception {
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest);
        Segment segment = ObjectBuilder.buildSegment();
        segment.setMetadata(ImmutableMap.of("A", "B"));
        segment.setContent("HELLO");
        Artifact artifact = a2ACmdCallableOnce.buildTextArtifact(segment);
        Assert.assertEquals(1, artifact.getParts().size());
        // 不指定Id和Meata
        Assert.assertNull(artifact.getArtifactId());
        Assert.assertNull(artifact.getMetadata());
        Part part = artifact.getParts().getFirst();
        Assert.assertEquals(part.getKind(), Part.TEXT_KIND);
        Assert.assertEquals(segment.getContent(), part.getText());
    }

    @Test
    public void testBuildDataArtifact() throws Exception {
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest);
        Segment segment = ObjectBuilder.buildSegment();
        segment.setMetadata(ImmutableMap.of("A", "B"));
        A2AData a2AData = new A2AData();
        a2AData.putAll(ImmutableMap.of("A", "B"));
        segment.setContent(JsonUtils.write(a2AData));
        Artifact artifact = a2ACmdCallableOnce.buildDataArtifact(a2AData.bindSegment(segment));
        Assert.assertEquals(1, artifact.getParts().size());
        // 不指定ID和Meta
        Assert.assertNull(artifact.getArtifactId());
        Assert.assertNull(artifact.getMetadata());
        Part part = artifact.getParts().getFirst();
        Assert.assertEquals(part.getKind(), Part.DATA_KIND);
        Assert.assertEquals("B", part.getData().get("A"));
    }

    @Test
    public void testBuildFailedTask1() throws Exception {
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest);
        Segment segment = ObjectBuilder.buildSegment();
        segment.setMetadata(ImmutableMap.of("A", "B"));
        segment.setContent("HELLO");
        Task task = a2ACmdCallableOnce.buildFailedTask(segment);
        Assert.assertEquals(task.getContextId(), a2ACmdCallableOnce.buildContextId());
        Assert.assertEquals(task.getId(), a2ACmdCallableOnce.buildTaskId());
        Assert.assertEquals(1, task.getArtifacts().size());
        Artifact artifact = task.getArtifacts().get(0);
        Assert.assertEquals(artifact.getMetadata(), segment.getMetadata());
        Assert.assertEquals(1, artifact.getParts().size());
        Assert.assertEquals(String.valueOf(segment.getIndex()), artifact.getArtifactId());
        Assert.assertEquals(segment.getMetadata(), artifact.getMetadata());
        Part part = artifact.getParts().getFirst();
        Assert.assertEquals(part.getKind(), Part.TEXT_KIND);
        Assert.assertEquals(segment.getContent(), part.getText());
        Assert.assertEquals(segment.getMetadata(), task.getMetadata());
        Assert.assertEquals(TaskStatus.STATUS_FAILED, task.getStatus().getState());
        Assert.assertNotNull(task.getTimestamp());
    }


    @Test
    public void testBuildFailedTask2() throws Exception {
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest);
        Task task = a2ACmdCallableOnce.buildFailedTask("HELLO");
        Assert.assertEquals(task.getContextId(), a2ACmdCallableOnce.buildContextId());
        Assert.assertEquals(task.getId(), a2ACmdCallableOnce.buildTaskId());
        Assert.assertEquals(1, task.getArtifacts().size());
        Artifact artifact = task.getArtifacts().get(0);
        Assert.assertEquals(1, artifact.getParts().size());
        Assert.assertEquals(36, artifact.getArtifactId().length());
        Assert.assertNull(artifact.getMetadata());
        Part part = artifact.getParts().getFirst();
        Assert.assertEquals(part.getKind(), Part.TEXT_KIND);
        Assert.assertEquals("HELLO", part.getText());
        Assert.assertNull(task.getMetadata());
        Assert.assertEquals(TaskStatus.STATUS_FAILED, task.getStatus().getState());
        Assert.assertNotNull(task.getTimestamp());
    }

    @Test
    public void testSuccessTaskWithDetailTask() throws Exception {
        Task task = Task.builder()
                .artifacts(Collections.singletonList(Artifact.builder()
                        .metadata(ImmutableMap.of("HELLO", "WORLD_1"))
                        .artifactId("MY ARTI")
                        .name("NAME")
                        .parts(List.of(Part.builder()
                                .text("HELLO")
                                .kind("TEXT")
                                .build()))
                        .build()))
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
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                Task _task = Task.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getId(), String.valueOf(this.a2aRequest.getId()));
                Assert.assertEquals(_task.getContextId(), this.a2aRequest.getTrace());
                Assert.assertEquals(_task.getTimestamp(), A2ACmdCallableOnce.FORMATTER.format(segment.getTimestamp()));
                Assert.assertEquals(_task.getStatus().getState(), TaskStatus.STATUS_COMPLETED);
                Assert.assertEquals(_task.getMetadata(), segment.getMetadata());
                Assert.assertEquals(1, _task.getArtifacts().size());
                Artifact artifact = _task.getArtifacts().getFirst();
                Assert.assertEquals("MY ARTI", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "WORLD_1");
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("HELLO", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
            }
        };
        a2ACmdCallableOnce.call(segment);
    }

    @Test
    public void testSuccessTaskWithDetailTaskAndMetadata() throws Exception {
        Task task = Task.builder()
                .metadata(ImmutableMap.of("HELLO_1", "HELLO_2", "HELLO", "HELLO_X"))
                .artifacts(Collections.singletonList(Artifact.builder()
                        .metadata(ImmutableMap.of("HELLO_1", "HELLO_2", "HELLO", "HELLO_X"))
                        .name("NAME")
                        .artifactId("ART_1")
                        .parts(List.of(Part.builder()
                                .text("HELLO")
                                .kind("TEXT")
                                .build()))
                        .build()))
                .build();
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD"))
                .content(new StringBuffer(JsonUtils.write(task)))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                Task _task = Task.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getId(), String.valueOf(this.a2aRequest.getId()));
                Assert.assertEquals(_task.getContextId(), this.a2aRequest.getTrace());
                Assert.assertEquals(_task.getTimestamp(), A2ACmdCallableOnce.FORMATTER.format(segment.getTimestamp()));
                Assert.assertEquals(_task.getStatus().getState(), TaskStatus.STATUS_COMPLETED);
                Assert.assertEquals(_task.getMetadata().get("HELLO_1"), "HELLO_2");
                // 不覆盖
                Assert.assertEquals(_task.getMetadata().get("HELLO"), "HELLO_X");
                Assert.assertEquals(1, _task.getArtifacts().size());
                Artifact artifact = _task.getArtifacts().getFirst();
                Assert.assertEquals("ART_1", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "HELLO_X");
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("HELLO", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
            }
        };
        a2ACmdCallableOnce.call(segment);
    }

    @Test
    public void testSuccessTaskWithDetailTaskWithDetail() throws Exception {
        Task task = Task.builder()
                .status(TaskStatus.builder()
                        .state(TaskStatus.STATUS_WORKING)
                        .build())
                .timestamp("ABCDE")
                .id("ID")
                .contextId("CONTEXT")
                .artifacts(Collections.singletonList(Artifact.builder()
                        .metadata(ImmutableMap.of("HELLO_1", "HELLO_2", "HELLO", "HELLO_X"))
                        .name("NAME")
                        .artifactId("ART_1")
                        .parts(List.of(Part.builder()
                                .text("HELLO")
                                .kind("TEXT")
                                .build()))
                        .build()))
                .build();
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD"))
                .content(new StringBuffer(JsonUtils.write(task)))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                Task _task = Task.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getId(), "ID");
                Assert.assertEquals(_task.getContextId(), "CONTEXT");
                Assert.assertEquals(_task.getTimestamp(), "ABCDE");
                Assert.assertEquals(_task.getStatus().getState(), TaskStatus.STATUS_WORKING);
                Assert.assertEquals(_task.getMetadata(), segment.getMetadata());
                Assert.assertEquals(1, _task.getArtifacts().size());
                Artifact artifact = _task.getArtifacts().getFirst();
                Assert.assertEquals("ART_1", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "HELLO_X");
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("HELLO", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
            }
        };
        a2ACmdCallableOnce.call(segment);
    }

    @Test
    public void testSuccessTaskWithDetailTaskWithMultiArtifact() throws Exception {
        Artifact artifact1 = Artifact.builder()
                .metadata(ImmutableMap.of("X", "Y"))
                .parts(List.of(Part.builder()
                        .kind("TEXT")
                        .text("DATA1")
                        .build(), Part.builder()
                        .kind("TEXT")
                        .text("DATA2")
                        .build()))
                .artifactId("ARTID_1")
                .build();
        Artifact artifact2 = Artifact.builder()
                .parts(List.of(Part.builder()
                        .kind("DATA")
                        .data(ImmutableMap.of("A", "B"))
                        .build()))
                .artifactId("ARTID_2")
                .build();
        Task task = Task.builder()
                .artifacts(List.of(artifact1, artifact2))
                .status(TaskStatus.builder()
                        .state(TaskStatus.STATUS_WORKING)
                        .build())
                .timestamp("ABCDE")
                .id("ID")
                .contextId("CONTEXT")
                .build();
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD"))
                .content(new StringBuffer(JsonUtils.write(task)))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                Task _task = Task.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getId(), "ID");
                Assert.assertEquals(_task.getContextId(), "CONTEXT");
                Assert.assertEquals(_task.getTimestamp(), "ABCDE");
                Assert.assertEquals(_task.getStatus().getState(), TaskStatus.STATUS_WORKING);
                Assert.assertEquals(_task.getMetadata(), segment.getMetadata());
                Assert.assertEquals(2, _task.getArtifacts().size());
                Artifact artifact = _task.getArtifacts().getFirst();
                Assert.assertEquals("ARTID_1", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("X"), "Y");
                Assert.assertEquals(2, artifact.getParts().size());
                Assert.assertEquals("DATA1", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
                Assert.assertEquals("DATA2", artifact.getParts().getLast().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getLast().getKind());
                artifact = _task.getArtifacts().getLast();
                Assert.assertEquals("ARTID_2", artifact.getArtifactId());
                Assert.assertNull(artifact.getMetadata());
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("B", artifact.getParts().getFirst().getData().get("A"));
                Assert.assertEquals("DATA", artifact.getParts().getFirst().getKind());
            }
        };
        a2ACmdCallableOnce.call(segment);
    }

    @Test
    public void testSuccessArtifact() throws Exception {
        Artifact artifact = Artifact.builder()
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
                .metadata(ImmutableMap.of("HELLO", "WORLD"))
                .content(new StringBuffer(JsonUtils.write(artifact)))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                Task _task = Task.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getId(), String.valueOf(this.a2aRequest.getId()));
                Assert.assertEquals(_task.getContextId(), this.a2aRequest.getTrace());
                Assert.assertEquals(_task.getTimestamp(), A2ACmdCallableOnce.FORMATTER.format(segment.getTimestamp()));
                Assert.assertEquals(_task.getStatus().getState(), TaskStatus.STATUS_COMPLETED);
                Assert.assertEquals(_task.getMetadata(), segment.getMetadata());
                // artifact部分
                Assert.assertEquals(1, _task.getArtifacts().size());
                Artifact artifact = _task.getArtifacts().getFirst();
                Assert.assertEquals("ARTID", artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata(), segment.getMetadata());
                Assert.assertEquals(2, artifact.getParts().size());
                Assert.assertEquals("DATA", artifact.getParts().getFirst().getText());
                Assert.assertEquals("TEXT", artifact.getParts().getFirst().getKind());
                Assert.assertEquals("B", artifact.getParts().getLast().getData().get("A"));
                Assert.assertEquals("DATA", artifact.getParts().getLast().getKind());
            }
        };
        a2ACmdCallableOnce.call(segment);
    }

    @Test
    public void testSuccessTaskWithSegment() throws Exception {
        Segment segment = ObjectBuilder.buildSegment(Segment.SegmentConfig.builder()
                .metadata(ImmutableMap.of("HELLO", "WORLD"))
                .content(new StringBuffer(JsonUtils.write(ImmutableMap.of("OK", "YES"))))
                .finished(false)
                .index(99)
                .code(222)
                .build());
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_response.json").openStream(), Map.class))
                .trace("TRACE")
                .build();
        MessageRequest messageRequest = new MessageRequest();
        A2ACmdCallableOnce a2ACmdCallableOnce = new A2ACmdCallableOnce(a2ARequest, messageRequest) {
            @Override
            protected void write(A2ACmdResponse a2ACmdResponse) throws Exception {
                Assert.assertEquals(a2ARequest.getId(), a2ACmdResponse.getId());
                // 永远200
                Assert.assertEquals(Integer.valueOf(200), a2ACmdResponse.getCode());
                Assert.assertEquals(segment.isFinished(), a2ACmdResponse.getFinished());
                Task _task = Task.class.cast(a2ACmdResponse.getResult());
                Assert.assertEquals(_task.getId(), "req-007");
                Assert.assertEquals(_task.getContextId(), "TRACE");
                Assert.assertNotNull(_task.getTimestamp());
                Assert.assertEquals(_task.getStatus().getState(), TaskStatus.STATUS_COMPLETED);
                Assert.assertEquals(_task.getMetadata(), segment.getMetadata());
                Assert.assertEquals(1, _task.getArtifacts().size());
                Artifact artifact = _task.getArtifacts().getFirst();
                Assert.assertEquals(String.valueOf(99), artifact.getArtifactId());
                Assert.assertEquals(artifact.getMetadata().get("HELLO"), "WORLD");
                Assert.assertEquals(1, artifact.getParts().size());
                Assert.assertEquals("YES", artifact.getParts().getFirst().getData().get("OK"));
                Assert.assertEquals("data", artifact.getParts().getFirst().getKind());
            }
        };
        a2ACmdCallableOnce.call(segment);
    }
    @Test
    public void testBuildTimestampNull() throws Exception {
        A2ACmdCallableOnce callable = new A2ACmdCallableOnce(null, null);
        Assert.assertNotNull(callable.buildTimestamp(System.currentTimeMillis()));
    }

    @Test
    public void testBuildSuccessTaskJsonParseError() throws Exception {
        A2ARequest a2aReq = EasyMock.createMock(A2ARequest.class);
        EasyMock.expect(a2aReq.getTrace()).andReturn("TRACE").anyTimes();
        EasyMock.expect(a2aReq.getId()).andReturn(1L).anyTimes();
        EasyMock.replay(a2aReq);
        A2ACmdCallableOnce callable = new A2ACmdCallableOnce(a2aReq, null);
        Segment segment = ObjectBuilder.buildSegment();
        segment.setContent("NOT_JSON");
        Task task = callable.buildSuccessTask(segment);
        Assert.assertEquals(1, task.getArtifacts().size());
        Assert.assertEquals("NOT_JSON", task.getArtifacts().get(0).getParts().get(0).getText());
    }
}
