package ai.open.right.workflow.flow.media.impl;

import ai.open.right.workflow.flow.media.MediaContext;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.Future;

public class MediaResourceImplTest {

    @Test
    public void testWithTransfer() throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream("HELLO WORLD".getBytes(StandardCharsets.UTF_8))).anyTimes();
        Future future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andReturn(response).anyTimes();
        EasyMock.replay(response, future, statusLine, entity);
        MediaContext mediaContext = new MediaContext();
        mediaContext.setData("HELLO WORLD");
        mediaContext.setType("IMAGE");
        MediaTransferServiceImpl.MediaHttpResource mediaResource = MediaTransferServiceImpl.MediaHttpResource.builder()
                .futureResponse(future)
                .mediaContext(mediaContext)
                .build();
        mediaResource.init();
        Assert.assertEquals("SEVMTE8gV09STEQ=", mediaContext.getData());
        Assert.assertEquals("inline:IMAGE", mediaContext.getType());
        EasyMock.verify(response, future, statusLine, entity);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithTransferWithStatus500() throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(statusLine.getStatusCode()).andReturn(500).anyTimes();
        Future future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andReturn(response).anyTimes();
        EasyMock.replay(response, future, statusLine);
        MediaContext mediaContext = new MediaContext();
        mediaContext.setData("HELLO WORLD");
        mediaContext.setType("IMAGE");
        MediaTransferServiceImpl.MediaHttpResource mediaResource = MediaTransferServiceImpl.MediaHttpResource.builder()
                .futureResponse(future)
                .mediaContext(mediaContext)
                .build();
        mediaResource.init();
        Assert.assertEquals("HELLO WORLD", mediaContext.getData());
        Assert.assertEquals("IMAGE", mediaContext.getType());
        EasyMock.verify(response, future, statusLine);
    }
}
