package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.listener.EventListenerService;
import ai.open.right.workflow.notify.NotifierService;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class GeminiRouterInitConfigTest {

    @Test
    public void testInit() throws Exception {
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        CloseableHttpAsyncClient stream = EasyMock.createMock(CloseableHttpAsyncClient.class);
        CloseableHttpAsyncClient once = EasyMock.createMock(CloseableHttpAsyncClient.class);
        Integer timeout = 10086;
        GeminiRouter.InitConfig initConfig = new GeminiRouter.InitConfig();
        initConfig.setEventListenerService(eventListenerService);
        initConfig.setNotifierService(notifierService);
        initConfig.setTimeout(timeout);
        initConfig.setStream(stream);
        initConfig.setOnce(once);
        initConfig.setUrlOnce("ONCE");
        initConfig.setUrlStream("STREAM");
        initConfig.setMaxSize(1024);
        GeminiRouter router = initConfig.geminiRouter();
        Assert.assertEquals(Integer.valueOf(1024), initConfig.getMaxSize());
        Assert.assertEquals("ONCE", router.getUrlOnce());
        Assert.assertEquals("STREAM", router.getUrlStream());
        Assert.assertEquals(eventListenerService, router.getEventListenerService());
        Assert.assertEquals(notifierService, router.getNotifierService());
        Assert.assertEquals(stream, router.getStream());
        Assert.assertEquals(once, router.getOnce());
        Assert.assertEquals(timeout, router.getTimeout());

    }
}
