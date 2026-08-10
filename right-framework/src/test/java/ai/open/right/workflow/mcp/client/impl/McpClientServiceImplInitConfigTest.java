package ai.open.right.workflow.mcp.client.impl;

import java.util.concurrent.ExecutorService;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.mcp.client.McpClientService;
import ai.open.right.workflow.notify.NotifierService;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.listener.EventListenerService;
import ai.open.right.resouce.PlaceholderResolver;

public class McpClientServiceImplInitConfigTest {

    @Test
    public void shouldCreateMcpClientServiceWithProvidedProperties() throws Exception {
        McpClientServiceImpl.InitConfig init = new McpClientServiceImpl.InitConfig();

        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        CloseableHttpAsyncClient httpClient = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EventListenerService eventListener = EasyMock.createMock(EventListenerService.class);
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);

        EasyMock.replay(placeholderResolver, httpClient, eventListener, executorService, notifierService, namesService);

        // 设置属性
        init.setPlaceholderResolver(placeholderResolver);
        init.setHttpClient(httpClient);
        init.setEventListener(eventListener);
        init.setExecutorService(executorService);
        init.setNotifierService(notifierService);
        init.setNamesService(namesService);
        init.setProcessor(4);
        init.setTimeout(120000);
        init.setBorrow(30000);
        McpClientServiceImpl bean = (McpClientServiceImpl) init.mcpClientService();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof McpClientService);

        EasyMock.verify(placeholderResolver, httpClient, eventListener, executorService, notifierService, namesService);
    }

    @Test
    public void shouldCreateMcpClientServiceWithDefaults() throws Exception {
        McpClientServiceImpl.InitConfig init = new McpClientServiceImpl.InitConfig();

        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        CloseableHttpAsyncClient httpClient = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EventListenerService eventListener = EasyMock.createMock(EventListenerService.class);
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        NotifierService notifierService = EasyMock.createMock(NotifierService.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);

        EasyMock.replay(placeholderResolver, httpClient, eventListener, executorService, notifierService, namesService);

        init.setPlaceholderResolver(placeholderResolver);
        init.setHttpClient(httpClient);
        init.setEventListener(eventListener);
        init.setExecutorService(executorService);
        init.setNotifierService(notifierService);
        init.setNamesService(namesService);

        McpClientServiceImpl bean = (McpClientServiceImpl) init.mcpClientService();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof McpClientService);

        EasyMock.verify(placeholderResolver, httpClient, eventListener, executorService, notifierService, namesService);
    }
}
