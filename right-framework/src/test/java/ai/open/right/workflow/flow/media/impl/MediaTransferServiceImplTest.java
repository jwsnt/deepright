package ai.open.right.workflow.flow.media.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

public class MediaTransferServiceImplTest {

    @Test
    public void testInit1() throws Exception {
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
        MediaTransferServiceImpl mediaTransfer = new MediaTransferServiceImpl() {
            protected Future<HttpResponse> getResponse(HttpRequestBase httpRequestBase) {
                return future;
            }
        };
        mediaTransfer.setResourceService(ObjectBuilder.buildResourceService());
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext each = new MediaContext();
        each.setType("IMAGE");
        each.setData("classpath:A2A.json");
        mediaContext.add(each);
        MediaConfig mediaConfig = new MediaConfig();
        mediaConfig.setBase64(true);
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        String expected = "WwogIHsKICAgICJuYW1lIjogInZpZGVvLWdlbmVyYXRvckBhMmEiLAogICAgImRlc2NyaXB0aW9uIjogIlByb3ZpZGVzIGFkdmFuY2VkIHJvdXRlIHBsYW5uaW5nLCB0cmFmZmljIGFuYWx5c2lzLCBhbmQgY3VzdG9tIG1hcCBnZW5lcmF0aW9uIHNlcnZpY2VzLiBUaGlzIGFnZW50IGNhbiBjYWxjdWxhdGUgb3B0aW1hbCByb3V0ZXMsIGVzdGltYXRlIHRyYXZlbCB0aW1lcyBjb25zaWRlcmluZyByZWFsLXRpbWUgdHJhZmZpYywgYW5kIGNyZWF0ZSBwZXJzb25hbGl6ZWQgbWFwcyB3aXRoIHBvaW50cyBvZiBpbnRlcmVzdC4iLAogICAgInByZWZlcnJlZFRyYW5zcG9ydCI6ICJKU09OUlBDIiwKICAgICJ2ZXJzaW9uIjogIjEuMi4wIiwKICAgICJkb2N1bWVudGF0aW9uVXJsIjogImh0dHBzOi8vZG9jcy5leGFtcGxlZ2Vvc2VydmljZXMuY29tL2dlb3JvdXRlLWFnZW50L2FwaSIsCiAgICAiY2FwYWJpbGl0aWVzIjogewogICAgICAic3RyZWFtaW5nIjogdHJ1ZSwKICAgICAgInB1c2hOb3RpZmljYXRpb25zIjogdHJ1ZSwKICAgICAgInN0YXRlVHJhbnNpdGlvbkhpc3RvcnkiOiBmYWxzZQogICAgfSwKICAgICJkZWZhdWx0SW5wdXRNb2RlcyI6IFsKICAgICAgImFwcGxpY2F0aW9uL2pzb24iLAogICAgICAidGV4dC9wbGFpbiIKICAgIF0sCiAgICAiZGVmYXVsdE91dHB1dE1vZGVzIjogWwogICAgICAiYXBwbGljYXRpb24vanNvbiIsCiAgICAgICJpbWFnZS9wbmciCiAgICBdLAogICAgInNraWxscyI6IFsKICAgICAgewogICAgICAgICJpZCI6ICJyb3V0ZS1vcHRpbWl6ZXItdHJhZmZpYyIsCiAgICAgICAgIm5hbWUiOiAiVHJhZmZpYy1Bd2FyZSBSb3V0ZSBPcHRpbWl6ZXIiLAogICAgICAgICJkZXNjcmlwdGlvbiI6ICJDYWxjdWxhdGVzIHRoZSBvcHRpbWFsIGRyaXZpbmcgcm91dGUgYmV0d2VlbiB0d28gb3IgbW9yZSBsb2NhdGlvbnMsIHRha2luZyBpbnRvIGFjY291bnQgcmVhbC10aW1lIHRyYWZmaWMgY29uZGl0aW9ucywgcm9hZCBjbG9zdXJlcywgYW5kIHVzZXIgcHJlZmVyZW5jZXMgKGUuZy4sIGF2b2lkIHRvbGxzLCBwcmVmZXIgaGlnaHdheXMpLiIsCiAgICAgICAgInRhZ3MiOiBbCiAgICAgICAgICAibWFwcyIsCiAgICAgICAgICAicm91dGluZyIsCiAgICAgICAgICAibmF2aWdhdGlvbiIsCiAgICAgICAgICAiZGlyZWN0aW9ucyIsCiAgICAgICAgICAidHJhZmZpYyIKICAgICAgICBdLAogICAgICAgICJleGFtcGxlcyI6IFsKICAgICAgICAgICJQbGFuIGEgcm91dGUgZnJvbSAnMTYwMCBBbXBoaXRoZWF0cmUgUGFya3dheSwgTW91bnRhaW4gVmlldywgQ0EnIHRvICdTYW4gRnJhbmNpc2NvIEludGVybmF0aW9uYWwgQWlycG9ydCcgYXZvaWRpbmcgdG9sbHMuIiwKICAgICAgICAgICJ7XCJvcmlnaW5cIjoge1wibGF0XCI6IDM3LjQyMiwgXCJsbmdcIjogLTEyMi4wODR9LCBcImRlc3RpbmF0aW9uXCI6IHtcImxhdFwiOiAzNy43NzQ5LCBcImxuZ1wiOiAtMTIyLjQxOTR9LCBcInByZWZlcmVuY2VzXCI6IFtcImF2b2lkX2ZlcnJpZXNcIl19IgogICAgICAgIF0KICAgICAgfQogICAgXQogIH0sCiAgewogICAgIm5hbWUiOiAidmlkZW8tZ2VuZXJhdG9yQGEyYV9zdHJlYW0iLAogICAgImRlc2NyaXB0aW9uIjogIlN0cmVhbSBQcm92aWRlcyBhZHZhbmNlZCByb3V0ZSBwbGFubmluZywgdHJhZmZpYyBhbmFseXNpcywgYW5kIGN1c3RvbSBtYXAgZ2VuZXJhdGlvbiBzZXJ2aWNlcy4gVGhpcyBhZ2VudCBjYW4gY2FsY3VsYXRlIG9wdGltYWwgcm91dGVzLCBlc3RpbWF0ZSB0cmF2ZWwgdGltZXMgY29uc2lkZXJpbmcgcmVhbC10aW1lIHRyYWZmaWMsIGFuZCBjcmVhdGUgcGVyc29uYWxpemVkIG1hcHMgd2l0aCBwb2ludHMgb2YgaW50ZXJlc3QuIiwKICAgICJwcmVmZXJyZWRUcmFuc3BvcnQiOiAiSlNPTlJQQyIsCiAgICAidmVyc2lvbiI6ICIxLjIuMCIsCiAgICAiZG9jdW1lbnRhdGlvblVybCI6ICJodHRwczovL2RvY3MuZXhhbXBsZWdlb3NlcnZpY2VzLmNvbS9nZW9yb3V0ZS1hZ2VudC1zdHJlYW0vYXBpIiwKICAgICJjYXBhYmlsaXRpZXMiOiB7CiAgICAgICJzdHJlYW1pbmciOiB0cnVlLAogICAgICAicHVzaE5vdGlmaWNhdGlvbnMiOiB0cnVlLAogICAgICAic3RhdGVUcmFuc2l0aW9uSGlzdG9yeSI6IHRydWUKICAgIH0sCiAgICAiZGVmYXVsdElucHV0TW9kZXMiOiBbCiAgICAgICJhcHBsaWNhdGlvbi9qc29uIgogICAgXSwKICAgICJkZWZhdWx0T3V0cHV0TW9kZXMiOiBbCiAgICAgICJpbWFnZS9wbmciCiAgICBdLAogICAgInNraWxscyI6IFsKICAgICAgewogICAgICAgICJpZCI6ICJTdHJlYW0gIHJvdXRlLW9wdGltaXplci10cmFmZmljIiwKICAgICAgICAibmFtZSI6ICJTdHJlYW0gIFRyYWZmaWMtQXdhcmUgUm91dGUgT3B0aW1pemVyIiwKICAgICAgICAiZGVzY3JpcHRpb24iOiAiU3RyZWFtICBDYWxjdWxhdGVzIHRoZSBvcHRpbWFsIGRyaXZpbmcgcm91dGUgYmV0d2VlbiB0d28gb3IgbW9yZSBsb2NhdGlvbnMsIHRha2luZyBpbnRvIGFjY291bnQgcmVhbC10aW1lIHRyYWZmaWMgY29uZGl0aW9ucywgcm9hZCBjbG9zdXJlcywgYW5kIHVzZXIgcHJlZmVyZW5jZXMgKGUuZy4sIGF2b2lkIHRvbGxzLCBwcmVmZXIgaGlnaHdheXMpLiIsCiAgICAgICAgInRhZ3MiOiBbCiAgICAgICAgICAibWFwcyIsCiAgICAgICAgICAicm91dGluZyIKICAgICAgICBdLAogICAgICAgICJleGFtcGxlcyI6IFsKICAgICAgICAgICJTdHJlYW0gcGxhbiBhIHJvdXRlIGZyb20gJzE2MDAgQW1waGl0aGVhdHJlIFBhcmt3YXksIE1vdW50YWluIFZpZXcsIENBJyB0byAnU2FuIEZyYW5jaXNjbyBJbnRlcm5hdGlvbmFsIEFpcnBvcnQnIGF2b2lkaW5nIHRvbGxzLiIsCiAgICAgICAgICAie1wib3JpZ2luXCI6IHtcImxhdFwiOiAzNy40MjIsIFwibG5nXCI6IC0xMjIuMDg0fSwgXCJkZXN0aW5hdGlvblwiOiB7XCJsYXRcIjogMzcuNzc0OSwgXCJsbmdcIjogLTEyMi40MTk0fSwgXCJwcmVmZXJlbmNlc1wiOiBbXCJhdm9pZF9mZXJyaWVzXCJdfSIKICAgICAgICBdCiAgICAgIH0KICAgIF0KICB9Cl0=";
        Assert.assertNotNull(mediaTransfer.getResourceService());
        Assert.assertEquals(expected, each.getData());
        Assert.assertEquals("inline:IMAGE", each.getType());
        EasyMock.verify(response, future, statusLine, entity);
    }

    @Test
    public void testInit2() throws Exception {
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
        MediaTransferServiceImpl mediaTransfer = new MediaTransferServiceImpl() {
            @Override
            protected Future<HttpResponse> getResponse(HttpRequestBase httpRequestBase, WorkflowTask workflowTask) {
                return future;
            }
        };
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext each = new MediaContext();
        each.setType("IMAGE");
        each.setData("http://1.2.3.com");
        mediaContext.add(each);
        MediaConfig mediaConfig = new MediaConfig();
        mediaConfig.setBase64(true);
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        Assert.assertEquals("SEVMTE8gV09STEQ=", each.getData());
        Assert.assertEquals("inline:IMAGE", each.getType());
        EasyMock.verify(response, future, statusLine, entity);
    }

    @Test
    public void testInit3() throws Exception {
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
        MediaTransferServiceImpl mediaTransfer = new MediaTransferServiceImpl() {
            protected Future<HttpResponse> getResponse(HttpRequestBase httpRequestBase) {
                return future;
            }
        };
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext each = new MediaContext();
        each.setType("IMAGE");
        each.setData("classpath:ABC.json");
        mediaContext.add(each);
        MediaConfig mediaConfig = new MediaConfig();
        mediaConfig.setBase64(true);
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        // 报错，保持原始值
        Assert.assertEquals("classpath:ABC.json", each.getData());
        Assert.assertEquals("IMAGE", each.getType());
        EasyMock.verify(response, future, statusLine, entity);
    }

    @Test
    public void testTransferWithDefaultConfig() throws Exception {
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
        MediaTransferServiceImpl mediaTransfer = new MediaTransferServiceImpl() {
            @Override
            protected Future<HttpResponse> getResponse(HttpRequestBase httpRequestBase, WorkflowTask workflowTask) {
                return future;
            }
        };
        mediaTransfer.init();
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext each = new MediaContext();
        each.setType("IMAGE");
        each.setData("http://1.2.3.com");
        mediaContext.add(each);
        mediaTransfer.transfer(ObjectBuilder.buildWorkflowTask(), mediaContext);
        Assert.assertEquals("SEVMTE8gV09STEQ=", each.getData());
        Assert.assertEquals("inline:IMAGE", each.getType());
        EasyMock.verify(response, future, statusLine, entity);
    }

    @Test
    public void testInitWithOutBase64() throws Exception {
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
        MediaTransferServiceImpl mediaTransfer = new MediaTransferServiceImpl() {
            protected Future<HttpResponse> getResponse(MediaContext mediaContext) {
                return future;
            }
        };
        List<MediaContext> mediaContext = new ArrayList<MediaContext>();
        MediaContext each = new MediaContext();
        each.setData("HELLO WORLD");
        each.setType("IMAGE");
        mediaContext.add(each);
        MediaConfig mediaConfig = new MediaConfig();
        mediaConfig.setBase64(false);
        mediaTransfer.transfer(mediaConfig, ObjectBuilder.buildWorkflowTask(), mediaContext);
        Assert.assertEquals("HELLO WORLD", each.getData());
        Assert.assertEquals("IMAGE", each.getType());
        EasyMock.verify(response, future, statusLine, entity);
    }

    @Test
    public void testGetResponse() throws Exception {
        HttpRequestBase httpRequestBase = new HttpGet("http://1.2.3.com");
        Future future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get(1000, TimeUnit.MILLISECONDS)).andReturn(null).anyTimes();
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.expect(client.execute(httpRequestBase, null)).andReturn(future).anyTimes();
        MediaTransferServiceImpl mediaTransfer = new MediaTransferServiceImpl();
        mediaTransfer.setResource(client);
        EasyMock.replay(client, future);
        Assert.assertEquals(future, mediaTransfer.getResponse(httpRequestBase, ObjectBuilder.buildWorkflowTask()));
        EasyMock.verify(client, future);
    }

    @Test
    public void testBuild() throws Exception {
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.replay(client);
        MediaTransferServiceImpl.InitConfig service = new MediaTransferServiceImpl.InitConfig();
        service.setResource(client);
        MediaTransferServiceImpl empty = (MediaTransferServiceImpl) service.mediaTransferService();
        Assert.assertEquals(client, empty.getResource());
        Assert.assertEquals(4, MediaTransferUtils.URI.size());
        EasyMock.verify(client);
    }
    @Test
    public void testIsNetworkInvalid() throws Exception {
        Assert.assertFalse(MediaTransferUtils.isNetwork("file:/12345"));
    }

    @Test(expected = IllegalArgumentException.class)
    public void testMediaHttpResourceFail() throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine status = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(response.getStatusLine()).andReturn(status).anyTimes();
        EasyMock.expect(status.getStatusCode()).andReturn(500).anyTimes();
        Future future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get()).andReturn(response).anyTimes();
        EasyMock.replay(response, status, future);
        MediaTransferServiceImpl.MediaHttpResource resource = MediaTransferServiceImpl.MediaHttpResource.builder()
            .futureResponse(future).mediaContext(new MediaContext()).build();
        resource.init(); // Should log error and return
    }
}
