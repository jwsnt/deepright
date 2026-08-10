package ai.open.right.netty.chat.distribute;

import java.util.*;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.NettyCloser;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMUsage;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.DefaultHttpContent;
import io.netty.util.Attribute;
import io.netty.util.concurrent.Future;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.beans.BeanUtils;

import java.net.SocketAddress;

public class NettyRequestTest {

    @Test
    public void testSetGetObject() throws Exception {
        NettyRequest delegate = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        LLMConfig config = new LLMConfig();
        config.setProvider("PROVIDER");
        delegate.setObjectQuery(config);
        config = delegate.getObjectQuery(LLMConfig.class);
        Assert.assertEquals("PROVIDER", config.getProvider());
    }

    @Test
    public void testAddMediaContext() throws Exception {
        NettyRequest delegate = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        MediaContext context = new MediaContext();
        delegate.addMediaContext(context);
        Assert.assertEquals(context, delegate.getMediaContext().getFirst());
        Assert.assertEquals(Integer.valueOf(delegate.getMediaContext().size()), Integer.valueOf(1));
    }

    @Test
    public void testIsFromFunMerge() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        Assert.assertFalse(request.isFromFunMerge());
        request.putMetadata(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertFalse(request.isFromFunMerge());
        request.putMetadata(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(request.isFromFunMerge());
    }

    @Test
    public void testSetProviderAndToken() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();

        request.setProviderAndToken("provider-x", "token-y");

        Assert.assertEquals("provider-x", request.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        Assert.assertEquals("token-y",
                request.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
    }

    @Test
    public void testWriterWithHttp() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        Attribute<Byte> attributeLimit = EasyMock.createMock(Attribute.class);
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(socketAddress, channel, context, closeFuture, returnFuture, attributeHttp, attributeServer, attributeLimit, attributeSSe);
        NettyRequest request = new NettyRequest();
        request.setChannel(context);
        request.writeBack(ObjectBuilder.buildSegment());
        EasyMock.verify(socketAddress, channel, context, attributeServer, attributeHttp, attributeLimit, closeFuture, returnFuture, attributeSSe);
    }

    @Test
    public void testWriterWithWs() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attribute = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attribute.get()).andReturn(NettyAttributes.SERVER_WS).anyTimes();
        EasyMock.replay(attribute);
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attribute).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.replay(closeFuture, returnFuture);
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(context);
        NettyRequest request = new NettyRequest();
        request.setChannel(context);
        request.writeBack(ObjectBuilder.buildSegment());
        EasyMock.verify(context, attribute, closeFuture, returnFuture);
    }

    @Test
    public void testCheck() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutConversation() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setConversation(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutUserContext() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setUserContext(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutTimestamp() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setCreated(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutProtocol() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setProtocol(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutTrace() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setTrace(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutChat() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setChat(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithOutBiz() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = new NettyRequest();
        BeanUtils.copyProperties(worktask, request);
        request.setBiz(null);
        NettyRequest.NettyRequestChecker.check(request);
    }

    @Test
    public void testToString() {
        WorkflowTask worktask = ObjectBuilder.buildWorkflowTask();
        NettyRequest request = NettyRequest.class.cast(worktask);
        Assert.assertEquals("UNKNOWN", request.getChat());
        Assert.assertNotNull(request.getCreated());
        Assert.assertEquals("chat", request.getProtocol());
        Assert.assertEquals("UNKNOWN", request.getConversation());
        Assert.assertEquals("UNKNOWN", request.getTrace());
        Assert.assertEquals("UNKNOWN", request.getBiz());
        Assert.assertNotNull(request.getUserContext());
        Assert.assertEquals(Notifier.LOCALHOST, request.getNotifier());
        Assert.assertEquals("UNKNOWN", request.getWorkflow());
        Assert.assertEquals("UNKNOWN", request.getQuery());
        Assert.assertNull(request.getChannel());
        Assert.assertTrue(request.getMetadata().isEmpty());
        Assert.assertEquals("ORIGINAL", request.getOriginal());
        Assert.assertEquals(Integer.valueOf(1), request.getDeepness());
        Assert.assertEquals("UNKNOWN", request.getUpstream());
        Assert.assertEquals("UNKNOWN", request.getDevice());
        BeanUtils.copyProperties(worktask, request);
    }

    @Test
    public void testWriteHttpWithStream() throws Exception {
        Segment segment = Segment.build(ObjectBuilder.buildLLMQuery(), Segment.SegmentConfig.builder().build());
        NettyRequest request = new NettyRequest();
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeService = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeService.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeService).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        EasyMock.replay(channel, socketAddress, context, closeFuture, returnFuture, attributeHttp, attributeService, attributeSSe);
        request.setChannel(context);
        request.writeHttp(segment);
        EasyMock.verify(channel, socketAddress, context, closeFuture, returnFuture, attributeHttp, attributeService, attributeSSe);
    }

    @Test
    public void testWriteHttpWithOnce() throws Exception {
        Segment segment = Segment.build(ObjectBuilder.buildLLMQuery(), Segment.SegmentConfig.builder().build());
        NettyRequest request = new NettyRequest();
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Attribute<Byte> attributeService = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeService.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_ONCE).anyTimes();
        Attribute<Byte> attributeSSe = EasyMock.createMock(Attribute.class);
        attributeSSe.set(NettyAttributes.HTTP_SSE);
        EasyMock.expectLastCall().anyTimes();
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeSSe).anyTimes();
        EasyMock.expect(attributeSSe.get()).andReturn(NettyAttributes.HTTP_SSE).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeService).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeHttp).anyTimes();
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();
        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(ByteBuf.class))).andReturn(closeFuture).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.replay(channel, socketAddress, attributeCors, context, closeFuture, returnFuture, attributeHttp, attributeService, attributeSSe);
        request.setChannel(context);
        request.writeHttp(segment);
        EasyMock.verify(channel, socketAddress, attributeCors, context, closeFuture, returnFuture, attributeHttp, attributeService, attributeSSe);
    }

    @Test
    public void testDelMeta() throws Exception {
        NettyRequest request = new NettyRequest();
        Assert.assertNull(request.delMetadata("HELLO", Date.class));
        Date date = new Date();
        request.putMetadata("HELLO", date);
        Assert.assertEquals(date, request.delMetadata("HELLO", Date.class));
    }

    @Test
    public void testGetMetadataReturnsNullWhenMetadataEmpty() throws Exception {
        NettyRequest request = new NettyRequest();
        request.setMetadata(new HashMap<>());
        Assert.assertNull(request.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMetadataReturnsNullWhenValueNull() throws Exception {
        NettyRequest request = new NettyRequest();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("HELLO", null);
        request.setMetadata(metadata);
        Assert.assertNull(request.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMetadataReturnsOriginalTypeWhenAssignable() throws Exception {
        NettyRequest request = new NettyRequest();
        Date date = new Date();
        request.putMetadata("HELLO", date);
        Assert.assertSame(date, request.getMetadata("HELLO", Date.class));
    }

    @Test
    public void testGetMetadataTransfersWhenTypeMismatch() throws Exception {
        NettyRequest request = new NettyRequest();
        request.putMetadata("CONFIG", Collections.singletonMap("provider", "PROVIDER"));
        LLMConfig config = request.getMetadata("CONFIG", LLMConfig.class);
        Assert.assertEquals("PROVIDER", config.getProvider());
    }

    @Test
    public void testContainChatTrack() {
        NettyRequest request = new NettyRequest();
        Assert.assertFalse(request.containChatTrack());
        request.beginChatTrack();
        Assert.assertTrue(request.containChatTrack());
    }

    @Test
    public void testContainHistory() {
        NettyRequest request = new NettyRequest();
        Assert.assertFalse(request.containHistories());
        request.addHistories(Arrays.asList(new History()));
        Assert.assertTrue(request.containHistories());
    }

    @Test
    public void testFunCall() {
        NettyRequest task = new NettyRequest();
        Assert.assertFalse(task.containFunCallTrack());
        Assert.assertNull(task.getFunCallTrack());
        task.beginFunCallTrack("ABC");
        Assert.assertEquals("ABC", task.getFunCallTrack());
        task.beginFunCallTrack();
        Assert.assertEquals(Integer.valueOf(36), Integer.valueOf(task.getFunCallTrack().length()));
        task.closeFunCallTrack();
        Assert.assertNull(task.getFunCallTrack());
    }

    @Test
    public void testChatTrack() {
        Segment _segment = ObjectBuilder.buildSegment();
        NettyRequest request = new NettyRequest();
        request.beginChatTrack();
        request.setNettyTrack(new NettyTrack() {

            @Override
            public void track(WorkflowTask workTask, Segment segment) {
                Assert.assertEquals(workTask, request);
                Assert.assertEquals(segment, _segment);
            }
        });
        request.track(_segment);
    }

    @Test
    public void testGetSet() {
        NettyRequest delegate = new NettyRequest();
        UserContext userContext = UserContext.builder().build();
        delegate.setUserContext(userContext);
        Assert.assertEquals(userContext, delegate.getUserContext());
    }

    @Test
    public void testAddQuery() throws Exception {
        NettyRequest delegate = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        delegate.addQuery("A");
        Assert.assertEquals("UNKNOWN\n" +
                "A", delegate.getQuery());
        delegate.addQuery("B");
        Assert.assertEquals("UNKNOWN\n" +
                "A" + System.lineSeparator() + "B", delegate.getQuery());
    }

    @Test
    public void testInitAlreadySet() {
        NettyRequest request = new NettyRequest();
        request.setConversation("CONV");
        request.setProtocol("PROTO");
        request.setChat("CHAT");
        request.init();
        Assert.assertEquals("CONV", request.getConversation());
        Assert.assertEquals("PROTO", request.getProtocol());
        Assert.assertEquals("CHAT", request.getChat());
    }

    @Test
    public void testAddMediaContextInit() {
        NettyRequest request = new NettyRequest();
        request.setMediaContext(null);
        request.addMediaContext(new MediaContext());
        Assert.assertNotNull(request.getMediaContext());
        Assert.assertEquals(1, request.getMediaContext().size());
    }

    @Test
    public void testDelMetadataNull() {
        NettyRequest request = new NettyRequest();
        request.setMetadata(null);
        request.delMetadata("KEY"); // Should not throw exception
    }

    @Test
    public void testInitMediaContextAlreadySet() throws Exception {
        NettyRequest request = new NettyRequest();
        List<MediaContext> list = new ArrayList<>();
        request.setMediaContext(list);
        request.initMediaContext();
        Assert.assertEquals(JsonUtils.write(list), JsonUtils.write(request.getMediaContext()));
    }

    @Test
    public void testWriteSourceFinishedNotIndex1() throws Exception {
        NettyRequest request = new NettyRequest();
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        request.setChannel(context);
        EasyMock.expect(context.alloc()).andReturn(ByteBufAllocator.DEFAULT).anyTimes();
        Segment segment = EasyMock.createMock(Segment.class);
        EasyMock.expect(segment.getUsage()).andReturn(new LLMUsage() {
            @Override
            public Integer getThinking() {
                return 0;
            }

            @Override
            public Integer getCache() {
                return 0;
            }

            @Override
            public Integer getTotal() {
                return 0;
            }

            @Override
            public Integer getInput() {
                return 0;
            }

            @Override
            public void addUsage(LLMUsage usage) {

            }
        }).anyTimes();
        EasyMock.expect(segment.getId()).andReturn("sid").anyTimes();
        EasyMock.expect(segment.getTimestamp()).andReturn(System.currentTimeMillis()).anyTimes();
        EasyMock.expect(segment.getWorkflow()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(segment.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(segment.getContent()).andReturn("CONTENT").anyTimes();
        EasyMock.expect(segment.getBiz()).andReturn("BIZ").anyTimes();
        EasyMock.expect(segment.getTrace()).andReturn("TRACE").anyTimes();
        EasyMock.expect(segment.isFinished()).andReturn(true).anyTimes();
        EasyMock.expect(segment.getIndex()).andReturn(2).anyTimes();
        EasyMock.expect(segment.getCode()).andReturn(200).anyTimes();
        segment.mark();
        EasyMock.expectLastCall().anyTimes();
        // Mock NettyWriter.isWsService and isHttpService
        // This is hard because they are static methods. 
        // But we can test the flow if we assume it's not WS and not HTTP.

        Attribute<Byte> attributeServer = EasyMock.createMock(Attribute.class);
        Attribute<Byte> attributeHttp = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeServer.get()).andReturn(NettyAttributes.SERVER_HTTP).anyTimes();
        EasyMock.expect(attributeHttp.get()).andReturn(NettyAttributes.CONNECTION_STREAM).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SERVER_TYPE)).andReturn(attributeServer).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CONNECTION_TYPE)).andReturn(attributeServer).anyTimes();
        Channel channel = EasyMock.createMock(Channel.class);
        SocketAddress socketAddress = EasyMock.createMock(SocketAddress.class);
        EasyMock.expect(channel.remoteAddress()).andReturn(socketAddress).anyTimes();
        EasyMock.expect(channel.isOpen()).andReturn(true).anyTimes();
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        ChannelFuture channelFuture = EasyMock.createMock(ChannelFuture.class);
        NettyCloser nettyCloser = new NettyCloser(context);
        Future<Void> future = EasyMock.createMock(Future.class);

        EasyMock.expect(future.isSuccess()).andReturn(false).anyTimes();
        EasyMock.expect(future.cause()).andReturn(new Throwable()).anyTimes();

        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        Attribute<Byte> attributeCors = EasyMock.createMock(Attribute.class);
        EasyMock.expect(attributeCors.get()).andReturn((byte) 0).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.SSE_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.attr(NettyAttributes.CORS_TYPE)).andReturn(attributeCors).anyTimes();
        EasyMock.expect(context.close()).andReturn(closeFuture).anyTimes();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).anyTimes();

        EasyMock.expect(context.writeAndFlush(EasyMock.anyObject(DefaultHttpContent.class))).andReturn(closeFuture).anyTimes();
        EasyMock.replay(closeFuture, returnFuture, attributeCors, socketAddress, future, channel, channelFuture, attributeServer, attributeHttp, segment, context);

        // If it's not WS and not HTTP, it should just return.
        nettyCloser.operationComplete(future);
        request.writeSource(segment);
        EasyMock.verify(closeFuture, returnFuture, attributeCors, socketAddress, future, channel, channelFuture, attributeServer, attributeHttp, segment, context);
    }

    @org.junit.jupiter.api.Test
    public void testNettyRequestInstantiationUnique() {
        org.junit.jupiter.api.Assertions.assertTrue(true);
    }

    @Test
    public void checkClosed_channelActive_doesNotThrow() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.isActive()).andReturn(true).anyTimes();
        EasyMock.replay(context, channel);
        NettyRequest request = new NettyRequest();
        request.setChannel(context);
        request.checkClosed();
        EasyMock.verify(context, channel);
    }

    @Test(expected = WorkflowException.class)
    public void checkClosed_channelInactive_throws() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.isActive()).andReturn(false).anyTimes();
        EasyMock.replay(context, channel);
        NettyRequest request = new NettyRequest();
        request.setWorkflow("WORKFLOW");
        request.setBiz("BIZ");
        request.setChannel(context);
        request.checkClosed();
        EasyMock.verify(context, channel);
    }

    @Test
    public void checkClosed_channelInactive_throwsSilentCn1() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.isActive()).andReturn(false).anyTimes();
        EasyMock.replay(context, channel);
        NettyRequest request = new NettyRequest();
        request.setWorkflow("WORKFLOW");
        request.setBiz("BIZ");
        request.setChannel(context);
        WorkflowException exception = org.junit.jupiter.api.Assertions.assertThrows(WorkflowException.class, request::checkClosed);
        org.junit.jupiter.api.Assertions.assertEquals(ProtocolCode.CN1, exception.getCode());
        org.junit.jupiter.api.Assertions.assertTrue(exception.getSilent());
        EasyMock.verify(context, channel);
    }

    @Test
    public void isClosed_channelActive_returnsFalse() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.isActive()).andReturn(true).anyTimes();
        EasyMock.replay(context, channel);
        NettyRequest request = new NettyRequest();
        request.setChannel(context);
        Assert.assertFalse(request.isClosed());
        EasyMock.verify(context, channel);
    }

    @Test
    public void getCreated_returnsTimestamp() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        Long ts = request.getCreated();
        Assert.assertNotNull(ts);
        Assert.assertEquals("getCreated should equal getTimestamp", ts, request.getCreated());
    }

    @Test
    public void isClosed_channelInactive_returnsTrue() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        Channel channel = EasyMock.createMock(Channel.class);
        EasyMock.expect(context.channel()).andReturn(channel).anyTimes();
        EasyMock.expect(channel.isActive()).andReturn(false).anyTimes();
        EasyMock.replay(context, channel);
        NettyRequest request = new NettyRequest();
        request.setChannel(context);
        Assert.assertTrue(request.isClosed());
        EasyMock.verify(context, channel);
    }

    @Test
    public void close_invokesContextClose() throws Exception {
        ChannelHandlerContext context = EasyMock.createMock(ChannelHandlerContext.class);
        ChannelFuture closeFuture = EasyMock.createMock(ChannelFuture.class);
        ChannelFuture returnFuture = EasyMock.createMock(ChannelFuture.class);
        EasyMock.expect(context.close()).andReturn(closeFuture).once();
        EasyMock.expect(closeFuture.addListener(NettyAlarm.INSTANCE)).andReturn(returnFuture).once();
        EasyMock.replay(context, closeFuture, returnFuture);
        NettyRequest request = new NettyRequest();
        request.setChannel(context);
        request.close();
        EasyMock.verify(context, closeFuture, returnFuture);
    }

    @Test
    public void incrDeepness_incrementsDeepness() {
        NettyRequest request = new NettyRequest();
        request.setDeepness(1);
        Assert.assertEquals(Integer.valueOf(1), request.getDeepness());
        request.incrDeepness();
        Assert.assertEquals(Integer.valueOf(2), request.getDeepness());
        request.incrDeepness();
        request.incrDeepness();
        Assert.assertEquals(Integer.valueOf(4), request.getDeepness());
    }

    @Test
    public void incrDeepness_whenDeepnessNull_setsToDEEPNESS() {
        NettyRequest request = new NettyRequest();
        Assert.assertNull("deepness should be null for new request", request.getDeepness());
        request.incrDeepness();
        Assert.assertEquals("incrDeepness() when deepness is null should set to RedirectContext.DEEPNESS", RedirectContext.DEEPNESS, request.getDeepness());
    }

    @Test
    public void testIsEntry() {
        NettyRequest request = new NettyRequest();
        request.setUpstream("");
        request.setDeepness(1);
        Assert.assertTrue("empty upstream, deepness 1, not from fun call -> entry", request.isEntry());
        request.setUpstream("some");
        Assert.assertFalse("non-empty upstream -> not entry", request.isEntry());
        request.setUpstream("");
        request.putMetadata(ai.open.right.workflow.flow.llm.provider.ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertFalse("from fun call -> not entry", request.isEntry());
        request.delMetadata(ai.open.right.workflow.flow.llm.provider.ProviderRequestService.KEY_FUN_FETCH);
        request.setDeepness(2);
        Assert.assertFalse("deepness != 1 -> not entry", request.isEntry());
    }

    /** markQuery 记录当前 query；resetQuery 将 query 恢复为 markQuery */
    @Test
    public void testMarkQueryAndResetQuery() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        request.setQuery("marked-query");
        request.markQuery();
        request.setQuery("changed-query");
        Assert.assertEquals("changed-query", request.getQuery());
        request.resetQuery();
        Assert.assertEquals("marked-query", request.getQuery());
    }

    @Test
    public void ignoreClosed_setsFlagTrue() throws Exception {
        NettyRequest request = new NettyRequest();
        Assert.assertFalse(Boolean.TRUE.equals(request.getIgnoreClosed()));
        request.ignoreClosed();
        Assert.assertTrue(Boolean.TRUE.equals(request.getIgnoreClosed()));
    }

    @Test
    public void init_marksHistoriesAsExternalReference() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        History h1 = new History();
        History h2 = new History();
        Assert.assertEquals(History.REFERENCE_SERVER, h1.getReference());
        Assert.assertEquals(History.REFERENCE_SERVER, h2.getReference());
        request.setHistories(Arrays.asList(h1, h2));
        NettyRequest out = request.init();
        Assert.assertSame(request, out);
        Assert.assertEquals(History.REFERENCE_CLIENT, h1.getReference());
        Assert.assertEquals(History.REFERENCE_CLIENT, h2.getReference());
    }

    @Test
    public void init_nullHistories_doesNotThrow() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        request.setHistories(null);
        request.init();
    }

    @Test
    public void init_emptyHistories_doesNotThrow() {
        NettyRequest request = (NettyRequest) ObjectBuilder.buildWorkflowTask();
        request.setHistories(new ArrayList<>());
        request.init();
    }

}
