package ai.open.right.workflow.a2a.server.impl;

import ai.open.right.netty.NettyAlarm;
import ai.open.right.netty.a2a.server.NettyA2ARequest;
import ai.open.right.netty.chat.server.NettyAttributes;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.server.A2ACmdExportService;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.ByteBufAllocator;
import io.netty.channel.Channel;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelHandlerContext;
import io.netty.util.Attribute;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.net.SocketAddress;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

public class A2ADistributorImplTest {

    @Test
    public void test() throws Exception {
        A2ARequest a2ARequest = NettyA2ARequest.builder().build();
        A2ACmdExportService a2ACmdExportService = EasyMock.createMock(A2ACmdExportService.class);
        EasyMock.expect(a2ACmdExportService.support(a2ARequest)).andReturn(true).anyTimes();
        a2ACmdExportService.cmd(a2ARequest);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(a2ACmdExportService);
        List<A2ACmdExportService> a2ACmdExportServiceList = new ArrayList<A2ACmdExportService>();
        a2ACmdExportServiceList.add(a2ACmdExportService);
        A2ADistributorImpl a2ADistributor = new A2ADistributorImpl();
        a2ADistributor.setA2aCmdExportService(a2ACmdExportServiceList);
        a2ADistributor.distribute(a2ARequest);
        EasyMock.verify(a2ACmdExportService);
    }

    @Test
    public void testEmpty() throws Exception {
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
                .context(context)
                .build();
        A2ADistributorImpl a2ADistributor = new A2ADistributorImpl();
        a2ADistributor.setA2aCmdExportService(new ArrayList<>());
        a2ADistributor.distribute(a2ARequest);
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture);
    }

    @Test
    public void testException() throws Exception {
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
                .context(context)
                .build();
        A2ACmdExportService a2ACmdExportService = EasyMock.createMock(A2ACmdExportService.class);
        EasyMock.expect(a2ACmdExportService.support(a2ARequest)).andThrow(new RuntimeException()).anyTimes();
        EasyMock.replay(a2ACmdExportService);
        A2ADistributorImpl a2ADistributor = new A2ADistributorImpl();
        a2ADistributor.setA2aCmdExportService(Arrays.asList(a2ACmdExportService));
        a2ADistributor.distribute(a2ARequest);
        EasyMock.verify(context, channel, socketAddress, attributeCors, closeFuture, returnFuture, a2ACmdExportService);
    }

    @Test
    public void testInit() throws Exception {
        List<A2ACmdExportService> a2ACmdExportServiceList = new ArrayList<A2ACmdExportService>();
        A2ADistributorImpl.InitConfig initConfig = new A2ADistributorImpl.InitConfig();
        initConfig.setA2aCmdExportService(a2ACmdExportServiceList);
        A2ADistributorImpl a2ADistributor = A2ADistributorImpl.class.cast(initConfig.a2aDistributor());
        Assert.assertEquals(a2ADistributor.getA2aCmdExportService(), initConfig.getA2aCmdExportService());
    }
    @Test
    public void testDistributeNullMethod() throws Exception {
        A2ADistributorImpl distributor = new A2ADistributorImpl();
        A2ARequest request = EasyMock.createMock(A2ARequest.class);
        EasyMock.expect(request.getMethod()).andReturn(null).anyTimes();
        EasyMock.expect(request.getId()).andReturn("1").anyTimes();
        request.writeOnce(EasyMock.anyObject());
        EasyMock.expectLastCall().once();
        EasyMock.replay(request);
        distributor.distribute(request);
        EasyMock.verify(request);
    }
}
