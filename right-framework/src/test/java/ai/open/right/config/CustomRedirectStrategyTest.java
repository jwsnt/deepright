package ai.open.right.config;

import org.apache.http.HttpRequest;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.protocol.HttpContext;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class CustomRedirectStrategyTest {

    @Test
    public void test() throws Exception {
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(statusLine.getStatusCode()).andReturn(307).anyTimes();
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        HttpRequest request = EasyMock.createMock(HttpRequest.class);
        HttpContext context = EasyMock.createMock(HttpContext.class);
        EasyMock.replay(response, request, context, statusLine);
        HttpClientConfig.CustomRedirectStrategy instance = HttpClientConfig.CustomRedirectStrategy.INSTANCE;
        Assert.assertTrue(instance.isRedirected(request, response, context));
        EasyMock.verify(response, request, context, statusLine);
    }
}
