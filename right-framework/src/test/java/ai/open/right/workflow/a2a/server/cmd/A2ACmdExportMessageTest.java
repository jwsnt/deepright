package ai.open.right.workflow.a2a.server.cmd;
import java.util.HashMap;
import ai.open.right.context.UserContext;
import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightService;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.protocol.MessageRequest;
import ai.open.right.workflow.sync.SyncCallable;
import io.netty.buffer.ByteBuf;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.net.SocketAddress;
import java.util.Map;

public class A2ACmdExportMessageTest {

    @Test
    public void testCmd() throws Exception {
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
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.expect(rightService.get(EasyMock.anyObject(RightConfig.class))).andReturn(null).anyTimes();
        EasyMock.replay(context, rightService, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_request.json").openStream(), Map.class))
                .context(context)
                .path("PATH/A@B")
                .build();
        A2ACmdExportMessage a2ACmdExportMessage = new A2ACmdExportMessage() {

            @Override
            protected RightConfig buildRightConfig(A2ARequest a2aRequest, MessageRequest messageRequest, SyncCallable syncCallable) throws Exception {
                RightConfig rightConfig = super.buildRightConfig(a2aRequest, messageRequest, syncCallable);
                Assert.assertEquals("{\"connectStream\":false,\"headers\":{},\"content\":{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"message/send\",\"params\":{\"message\":{\"role\":\"user\",\"parts\":[{\"kind\":\"text\",\"text\":\"tell me a joke1\"}],\"messageId\":\"9229e770-767c-417b-a0b0-f0741243c589\"}}},\"path\":\"PATH/A@B\",\"method\":\"MESSAGE/SEND\",\"id\":1}", JsonUtils.write(a2aRequest));
                Assert.assertEquals("{\"metadata\":{},\"message\":{\"parts\":[{\"text\":\"tell me a joke1\",\"kind\":\"text\"}],\"messageId\":\"9229e770-767c-417b-a0b0-f0741243c589\",\"kind\":\"message\",\"role\":\"user\"}}", JsonUtils.write(messageRequest));
                Assert.assertNull(syncCallable);
                return rightConfig;
            }

            @Override
            public SyncCallable buildSyncCallable(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
                return null;
            }

            @Override
            public Boolean support(A2ARequest a2aRequest) throws Exception {
                return null;
            }
        };
        a2ACmdExportMessage.setRightService(rightService);
        a2ACmdExportMessage.setTimeout4Llm(1024);
        a2ACmdExportMessage.cmd(a2ARequest);
        EasyMock.verify(context, rightService, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testMedia() throws Exception {
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
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.expect(rightService.get(EasyMock.anyObject(RightConfig.class))).andReturn(null).anyTimes();
        EasyMock.replay(context, rightService, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithFilePart_request.json").openStream(), Map.class))
                .context(context)
                .path("PATH/A@B")
                .build();
        A2ACmdExportMessage a2ACmdExportMessage = new A2ACmdExportMessage() {

            @Override
            protected RightConfig buildRightConfig(A2ARequest a2aRequest, MessageRequest messageRequest, SyncCallable syncCallable) throws Exception {
                RightConfig rightConfig = super.buildRightConfig(a2aRequest, messageRequest, syncCallable);
                Assert.assertEquals("{\"connectStream\":false,\"headers\":{},\"content\":{\"jsonrpc\":\"2.0\",\"id\":\"req-007\",\"method\":\"message/send\",\"params\":{\"message\":{\"role\":\"user\",\"parts\":[{\"kind\":\"text\",\"text\":\"Analyze this image and highlight any faces.\"},{\"kind\":\"file\",\"file\":{\"name\":\"input_image.png\",\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"#\":\"Base64 encoded image data\"}}],\"messageId\":\"6dbc13b5-bd57-4c2b-b503-24e381b6c8d6\"}}},\"path\":\"PATH/A@B\",\"method\":\"MESSAGE/SEND\",\"id\":\"req-007\"}", JsonUtils.write(a2aRequest));
                Assert.assertEquals("{\"metadata\":{},\"message\":{\"parts\":[{\"text\":\"Analyze this image and highlight any faces.\",\"kind\":\"text\"},{\"file\":{\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"name\":\"input_image.png\",\"content\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\"},\"kind\":\"file\"}],\"messageId\":\"6dbc13b5-bd57-4c2b-b503-24e381b6c8d6\",\"kind\":\"message\",\"role\":\"user\"}}", JsonUtils.write(messageRequest));
                Assert.assertNull(syncCallable);
                return rightConfig;
            }

            @Override
            public SyncCallable buildSyncCallable(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
                return null;
            }

            @Override
            public Boolean support(A2ARequest a2aRequest) throws Exception {
                return null;
            }
        };
        a2ACmdExportMessage.setRightService(rightService);
        a2ACmdExportMessage.setTimeout4Llm(1024);
        a2ACmdExportMessage.cmd(a2ARequest);
        EasyMock.verify(context, rightService, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testMediaWithData() throws Exception {
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
        RightService rightService = EasyMock.createMock(RightService.class);
        EasyMock.expect(rightService.get(EasyMock.anyObject(RightConfig.class))).andReturn(null).anyTimes();
        EasyMock.replay(context, rightService, channel, socketAddress, attributeCors, closeFuture, returnFuture);
        A2ARequest a2ARequest = NettyA2ARequest.builder()
                .content(JsonUtils.read(ResourceUtils.getURL("classpath:A2A_MessageSend_WithDataPart_request.json").openStream(), Map.class))
                .context(context)
                .path("PATH/A@B")
                .build();
        A2ACmdExportMessage a2ACmdExportMessage = new A2ACmdExportMessage() {

            @Override
            protected RightConfig buildRightConfig(A2ARequest a2aRequest, MessageRequest messageRequest, SyncCallable syncCallable) throws Exception {
                RightConfig rightConfig = super.buildRightConfig(a2aRequest, messageRequest, syncCallable);
                Assert.assertEquals("{\"connectStream\":false,\"headers\":{},\"content\":{\"jsonrpc\":\"2.0\",\"id\":\"req-007\",\"method\":\"message/send\",\"params\":{\"message\":{\"role\":\"user\",\"parts\":[{\"kind\":\"text\",\"text\":\"Analyze this image and highlight any faces.\"},{\"kind\":\"data\",\"data\":{\"name\":\"input_image.png\",\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"#\":\"Base64 encoded image data\"}},{\"kind\":\"error\",\"data\":{\"name\":\"input_image.png\",\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"#\":\"Base64 encoded image data\"}}],\"messageId\":\"6dbc13b5-bd57-4c2b-b503-24e381b6c8d6\"}}},\"path\":\"PATH/A@B\",\"method\":\"MESSAGE/SEND\",\"id\":\"req-007\"}", JsonUtils.write(a2aRequest));
                Assert.assertEquals("{\"metadata\":{},\"message\":{\"parts\":[{\"text\":\"Analyze this image and highlight any faces.\",\"kind\":\"text\"},{\"data\":{\"name\":\"input_image.png\",\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"#\":\"Base64 encoded image data\"},\"kind\":\"data\"},{\"data\":{\"name\":\"input_image.png\",\"mimeType\":\"image/png\",\"bytes\":\"iVBORw0KGgoAAAANSUhEUgAAAAUA...\",\"#\":\"Base64 encoded image data\"},\"kind\":\"error\"}],\"messageId\":\"6dbc13b5-bd57-4c2b-b503-24e381b6c8d6\",\"kind\":\"message\",\"role\":\"user\"}}", JsonUtils.write(messageRequest));
                Assert.assertNull(syncCallable);
                return rightConfig;
            }

            @Override
            public SyncCallable buildSyncCallable(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
                return null;
            }

            @Override
            public Boolean support(A2ARequest a2aRequest) throws Exception {
                return null;
            }
        };
        a2ACmdExportMessage.setRightService(rightService);
        a2ACmdExportMessage.setTimeout4Llm(1024);
        a2ACmdExportMessage.cmd(a2ARequest);
        EasyMock.verify(context, rightService, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }
    @Test
    public void testBuildUserContextEmpty() throws Exception {
        A2ACmdExportMessage service = new A2ACmdExportMessage() {
            @Override public SyncCallable buildSyncCallable(A2ARequest a, MessageRequest m) { return null; }
            @Override public Boolean support(A2ARequest a) { return true; }
        };
        MessageRequest msgReq = new MessageRequest();
        msgReq.setMetadata(new HashMap<>());
        UserContext uc = service.buildUserContext(null, msgReq);
        Assert.assertNotNull(uc);
    }

    /**
     * 覆盖 A2ANotifierWriteBack：构造、writeSource（非 JSON 与 JSON 分支）、writeBack、buildA2Response、buildTask、buildArtifact、buildPart
     */
    @Test
    public void testA2ANotifierWriteBackNonJson() throws Exception {
        A2ARequest request = EasyMock.createMock(A2ARequest.class);
        EasyMock.expect(request.getId()).andReturn(1L).anyTimes();
        request.writeStream(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        A2ACmdExportMessage.A2ANotifierWriteBack wb = new A2ACmdExportMessage.A2ANotifierWriteBack(request);
        Assert.assertSame(request, wb.getRequest());
        ai.open.right.workflow.flow.llm.Segment segment = ai.open.right.ObjectBuilder.buildSegment();
        segment.setContent("plain text");
        wb.writeSource(segment);
        wb.writeBack(segment);
        EasyMock.verify(request);
    }

    @Test
    public void testA2ANotifierWriteBackJson() throws Exception {
        A2ARequest request = EasyMock.createMock(A2ARequest.class);
        request.writeStream(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(request);
        A2ACmdExportMessage.A2ANotifierWriteBack wb = new A2ACmdExportMessage.A2ANotifierWriteBack(request);
        ai.open.right.workflow.flow.llm.Segment segment = ai.open.right.ObjectBuilder.buildSegment();
        segment.setContent("{\"id\":1,\"finished\":true,\"result\":{}}");
        wb.writeSource(segment);
        EasyMock.verify(request);
    }
}
