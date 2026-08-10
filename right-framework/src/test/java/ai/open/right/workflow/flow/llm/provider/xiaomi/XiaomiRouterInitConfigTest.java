package ai.open.right.workflow.flow.llm.provider.xiaomi;

import ai.open.right.listener.EventListenerService;
import ai.open.right.workflow.notify.NotifierService;
import org.apache.http.concurrent.FutureCallback;
import org.apache.http.nio.protocol.HttpAsyncRequestProducer;
import org.apache.http.nio.protocol.HttpAsyncResponseConsumer;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.protocol.HttpContext;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.IOException;
import java.util.concurrent.Future;

public class XiaomiRouterInitConfigTest {

    @Test
    public void testInit() throws Exception {
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        CloseableHttpAsyncClient stream = new EmptyCloseableHttpAsyncClient();
        CloseableHttpAsyncClient once = new EmptyCloseableHttpAsyncClient();

        XiaomiRouter.InitConfig initConfig = new XiaomiRouter.InitConfig();
        initConfig.setEventListenerService(eventListenerService);
        initConfig.setNotifierService(notifierService);
        initConfig.setTimeout(10086);
        initConfig.setStream(stream);
        initConfig.setOnce(once);
        initConfig.setUrl("URL");
        initConfig.setMaxSize(1024);

        XiaomiRouter router = initConfig.xiaomiRouter();

        Assert.assertEquals("URL", router.getUrl());
        Assert.assertEquals(Integer.valueOf(1024), initConfig.getMaxSize());
        Assert.assertEquals(Integer.valueOf(10086), router.getTimeout());
        Assert.assertEquals(eventListenerService, router.getEventListenerService());
        Assert.assertEquals(notifierService, router.getNotifierService());
        Assert.assertEquals(stream, router.getStream());
        Assert.assertEquals(once, router.getOnce());
    }

    private static final class EmptyCloseableHttpAsyncClient extends CloseableHttpAsyncClient {

        @Override
        public boolean isRunning() {
            return false;
        }

        @Override
        public void start() {
        }

        @Override
        public void close() throws IOException {
        }

        @Override
        public <T> Future<T> execute(HttpAsyncRequestProducer requestProducer, HttpAsyncResponseConsumer<T> responseConsumer, FutureCallback<T> callback) {
            return null;
        }

        @Override
        public <T> Future<T> execute(HttpAsyncRequestProducer requestProducer, HttpAsyncResponseConsumer<T> responseConsumer, HttpContext context, FutureCallback<T> callback) {
            return null;
        }
    }
}
