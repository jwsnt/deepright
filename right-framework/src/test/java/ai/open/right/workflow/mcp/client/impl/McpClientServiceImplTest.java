package ai.open.right.workflow.mcp.client.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.listener.Event;
import ai.open.right.listener.EventListenerService;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.McpClient;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.pool2.BasePooledObjectFactory;
import org.apache.commons.pool2.PooledObject;
import org.apache.commons.pool2.impl.DefaultPooledObject;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.apache.commons.pool2.impl.GenericObjectPoolConfig;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.CollectionUtils;
import org.springframework.util.ResourceUtils;
import org.springframework.util.StringUtils;

import java.io.File;
import java.util.*;
import java.util.concurrent.*;

public class McpClientServiceImplTest {

    @Test
    public void testPython() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesService namesService = ObjectBuilder.buildNamesService();
        mcpClientService.setNamesService(namesService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        Assert.assertEquals(executorService, mcpClientService.getExecutorService());
        String expect = "[{\"properties\":{\"sql\":{\"type\":\"string\"}},\"required\":[\"sql\"],\"description\":\"Execute SQL queries safely\",\"refer\":false,\"name\":\"Tools_DCxDzTDhRgCghRTjCyyTQDBSAjSRxwDj\"}]";
        McpDimension mcpDimension = McpDimension.builder().build();
        Assert.assertEquals(expect, JsonUtils.write(mcpClientService.toolsList("SQLite Explorer", mcpDimension)));
        try {
            String encode = namesService.encode(NamesService.PREFIX_TOOLS, "SQLite Explorer", "query_data");
            mcpClientService.toolsCall(encode, Collections.singletonMap("sql", "SELECT 1"), mcpDimension);
        } catch (Exception e) {
            Assert.assertEquals("Output validation error: {'A': 'B', 'C': {'D': 'E'}} is not of type 'string'", e.getMessage());
        }
        String expected = "[{\"properties\":{\"text\":{\"description\":\"\",\"type\":\"string\"}},\"required\":[\"text\"],\"description\":\"Generate a prompt asking for a summary.\",\"refer\":false,\"name\":\"Prompt_CQyjziyRhwCQjxSiyCChgBgTghCiCAgQ\"}]";
        Assert.assertEquals(expected, JsonUtils.write(mcpClientService.promptList("SQLite Explorer", mcpDimension)));
        expected = "[{\"properties\":{\"uri\":{\"description\":\"URI format: schema://main. Provide the database schema as a resource\",\"type\":\"string\"}},\"required\":[\"uri\"],\"description\":\"URI format: schema://main. Provide the database schema as a resource\",\"refer\":false,\"name\":\"Resource_gzgzhBRQCDjjjzgTyhDTBziByQAyzByj\"}]";
        Assert.assertEquals(expected, JsonUtils.write(mcpClientService.resourcesList("SQLite Explorer", mcpDimension)));
        Assert.assertEquals("A\n" + "B\n", mcpClientService.resourcesRead("SQLite Explorer", "schema://main", null, mcpDimension).getResult());
        Assert.assertEquals("Please summarize the following text:\n" + "\n" + "MyTable", mcpClientService.promptGet("SQLite Explorer", "summarize_request", Collections.singletonMap("text", "MyTable"), null, mcpDimension).getResult().getFirst().getContent());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
        new File("sqlite3_database.db").deleteOnExit();
    }

    @Test
    public void testNpxWithToolsListException() throws Exception {
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesServiceImpl namesService = new NamesServiceImpl() {
            @Override
            public String encode(String input) throws Exception {
                if (StringUtils.endsWithIgnoreCase("secure-filesystem-server__move_file", input)) {
                    throw new RuntimeException();
                }
                return super.encode(input);
            }
        };
        namesService.init();
        mcpClientService.setNamesService(namesService);
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setTimeout(10000);
        mcpClientService.setProcessor(1);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        McpDimension mcpDimension = McpDimension.builder().build();
        Assert.assertEquals(Integer.valueOf(10), Integer.valueOf(mcpClientService.toolsList("secure-filesystem-server", mcpDimension).size()));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testNpx() throws Exception {
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setTimeout(10000);
        mcpClientService.setProcessor(10);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        McpDimension mcpDimension = McpDimension.builder().build();
        Assert.assertNotNull(JsonUtils.write(mcpClientService.toolsList("secure-filesystem-server", mcpDimension)));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test(expected = NoSuchMethodException.class)
    public void testFailed() throws Exception {
        GenericObjectPool pool = EasyMock.createMock(GenericObjectPool.class);
        McpClient mcpClient = EasyMock.createMock(McpClient.class);
        EasyMock.expect(pool.borrowObject(100L)).andReturn(mcpClient).anyTimes();
        pool.invalidateObject(mcpClient);
        EasyMock.replay(pool, mcpClient);
        McpClientServiceImpl.McpFuture mcpFuture = new McpClientServiceImpl.McpFuture(pool, 100, "HELLO", null);
        try {
            mcpFuture.call();
        } finally {
            EasyMock.verify(pool, mcpClient);
        }
    }

    @Test
    public void testPythonWithEmptyTools() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        Assert.assertEquals(executorService, mcpClientService.getExecutorService());
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> funCalls = mcpClientService.toolsList("empty-server", mcpDimension);
        Assert.assertTrue(funCalls.isEmpty());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGet1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet("echo", "echo_prompt", map, null, mcpDimension);
        Assert.assertTrue(!history.getResult().isEmpty());
        Assert.assertEquals("My name hello, introduce me", history.getResult().getFirst().getContent());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGet2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        NamesService mcpNameService = ObjectBuilder.buildNamesService();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(mcpNameService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        String client = mcpNameService.encode(NamesService.PREFIX_PROMPT, "echo", "echo_prompt");
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet(client, map, mcpDimension);
        Assert.assertTrue(!history.getResult().isEmpty());
        Assert.assertEquals("My name hello, introduce me", history.getResult().getFirst().getContent());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGet3() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> result = mcpClientService.promptGet("echo", "echo_prompt_", map, null, mcpDimension);
        Assert.assertTrue(CollectionUtils.isEmpty(result.getResult()));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGet4() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        NamesService mcpNameService = ObjectBuilder.buildNamesService();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO WORLD"));
        mcpClientService.setNamesService(mcpNameService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet("echo", null, null, McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).build(), mcpDimension);
        Assert.assertTrue(history.getResult().isEmpty());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGetWithHistory() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(""));
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(1);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(10000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet("echo", null, map, McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).build(), mcpDimension);
        Assert.assertEquals(McpClientServiceImpl.EMPTY_HISTORY, history.getResult());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGetWithEmtpyHistory() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(""));
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(""));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet("echo", null, map, McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).build(), mcpDimension);
        Assert.assertTrue(history.getResult().isEmpty());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGetWithList() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        List<History> histories = new ArrayList<>();
        History h1 = new History();
        h1.setRole(History.ROLE_USER);
        h1.setContent("WORLD");
        histories.add(h1);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(JsonUtils.write(Arrays.asList(h1))));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet("echo", null, map, McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).dynamic("DYNAMIC").timeout(1000).build(), mcpDimension);
        Assert.assertTrue(!history.getResult().isEmpty());
        Assert.assertEquals("WORLD", history.getResult().getFirst().getContent());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptGetWithListWithException() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        List<History> histories = new ArrayList<>();
        History h1 = new History();
        h1.setRole(History.ROLE_USER);
        histories.add(h1);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(JsonUtils.write(Arrays.asList(h1))));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> history = mcpClientService.promptGet("echo", "text", map, McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).dynamic("DYNAMIC").timeout(1000).build(), mcpDimension);
        Assert.assertTrue(history.getResult().isEmpty());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testPromptListWithException() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(new NamesServiceImpl() {
            @Override
            public String encode(String prefix, String client, String name) {
                throw new RuntimeException();
            }
        });
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        HashMap map = new HashMap<>();
        map.put("message", "hello");
        List<History> histories = new ArrayList<>();
        History h1 = new History();
        h1.setRole(History.ROLE_USER);
        histories.add(h1);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(JsonUtils.write(Arrays.asList(h1))));
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> history = mcpClientService.promptList("echo", mcpDimension);
        Assert.assertTrue(history.isEmpty());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testWithPromptListEmpty() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        Assert.assertEquals(executorService, mcpClientService.getExecutorService());
        McpDimension mcpDimension = McpDimension.builder().build();
        Assert.assertTrue(mcpClientService.promptList("test", mcpDimension).isEmpty());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceListWithException() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(new NamesServiceImpl() {
            @Override
            public String encode(String prefix, String client, String name) throws Exception {
                if (StringUtils.endsWithIgnoreCase(name, "schema://main")) {
                    throw new RuntimeException();
                }
                return super.encode(prefix, client, name);
            }
        });
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(10000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> providerFunCalls = mcpClientService.resourcesList("empty-server", mcpDimension);
        Assert.assertEquals(Integer.valueOf(0), Integer.valueOf(providerFunCalls.size()));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceList1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(10000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> providerFunCalls = mcpClientService.resourcesList("empty-server", mcpDimension);
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(providerFunCalls.size()));
        Assert.assertEquals("Resource_BiARQxhRABiRBiQCDygTRSTgizzzTzgD", providerFunCalls.getFirst().getName());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test(expected = ExecutionException.class)
    public void testResourceList2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        try {
            McpDimension mcpDimension = McpDimension.builder().build();
            mcpClientService.resourcesList("secure-filesystem-server", mcpDimension);
        } finally {
            mcpClientService.destroy();
            executorService.shutdown();
            EasyMock.verify(eventListenerService);
        }
    }

    @Test
    public void testResourceList3() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> providerFunCalls = mcpClientService.resourcesList("schema-server", mcpDimension);
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(providerFunCalls.size()));
        Assert.assertEquals("Resource_SyCAzzQhRCCSTRzxTAwQwAgBRgwACzyi", providerFunCalls.getFirst().getName());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceListWithEmpty() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> providerFunCalls = mcpClientService.resourcesList("multi_roles_prompt", mcpDimension);
        Assert.assertEquals(Integer.valueOf(0), Integer.valueOf(providerFunCalls.size()));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourcesTemplatesList() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> providerFunCalls = mcpClientService.resourcesTemplatesList("dynamic_resource", mcpDimension);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(providerFunCalls.size()));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourcesTemplatesListWithException() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesServiceImpl namesService = new NamesServiceImpl() {
            @Override
            public String encode(String prefix, String client, String name) throws Exception {
                if (StringUtils.endsWithIgnoreCase(name, "db://users/{user_id}/email")) {
                    throw new RuntimeException();
                }
                return super.encode
                        (prefix, client, name);
            }
        };
        namesService.init();
        mcpClientService.setNamesService(namesService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(10000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        McpDimension mcpDimension = McpDimension.builder().build();
        List<ProviderFunCall> providerFunCalls = mcpClientService.resourcesTemplatesList("dynamic_resource", mcpDimension);
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(providerFunCalls.size()));
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceRead1() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<String> resourcesRead = mcpClientService.resourcesRead("empty-server", "schema://main", McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).dynamic("DYNAMIC").timeout(1000).build(), mcpDimension);
        Assert.assertEquals("A\n" + "B\n", resourcesRead.getResult());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceRead2() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<String> resourcesRead = mcpClientService.resourcesRead("empty-server", null, McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).dynamic("DYNAMIC").timeout(1000).build(), mcpDimension);
        Assert.assertEquals("WORLD", resourcesRead.getResult());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceRead3() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesService mcpNameService = ObjectBuilder.buildNamesService();
        mcpClientService.setNamesService(mcpNameService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<String> resourcesRead = mcpClientService.resourcesRead(mcpNameService.encode(NamesService.PREFIX_RESOURCE, "empty-server", "schema://main"), "schema://main", mcpDimension);
        Assert.assertEquals("A\n" + "B\n", resourcesRead.getResult());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testResourceRead4() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        mcpClientService.setNamesService(ObjectBuilder.buildNamesService());
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<String> resourcesRead = mcpClientService.resourcesRead("empty-server", "schema://master", McpRuntime.builder().workTask(ObjectBuilder.buildWorkflowTask()).dynamic("DYNAMIC").timeout(1000).build(), mcpDimension);
        Assert.assertEquals("", resourcesRead.getResult());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test
    public void testMultiRolesPrompt() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesService mcpNameService = ObjectBuilder.buildNamesService();
        mcpClientService.setNamesService(mcpNameService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        McpDimension mcpDimension = McpDimension.builder().build();
        McpResult<List<History>> histories = mcpClientService.promptGet("multi_roles_prompt", "debug_session_start", Collections.singletonMap("error_message", "HELLO WORLD"), null, mcpDimension);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(histories.getResult().size()));
        Assert.assertEquals(History.ROLE_USER, histories.getResult().getFirst().getRole());
        Assert.assertEquals(History.ROLE_ASSISTANT, histories.getResult().get(1).getRole());
        Assert.assertEquals("your name is:\n" + "HELLO WORLD", histories.getResult().getFirst().getContent());
        Assert.assertEquals("Okay, I can help with that. Can you provide the full traceback and tell me what you were trying to do?", histories.getResult().get(1).getContent());
        mcpClientService.destroy();
        executorService.shutdown();
        EasyMock.verify(eventListenerService);
    }

    @Test(expected = ExecutionException.class)
    public void testHttpStreamingWithErrorNetWork() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesService mcpNameService = ObjectBuilder.buildNamesService();
        mcpClientService.setNamesService(mcpNameService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        try {
            McpDimension mcpDimension = McpDimension.builder().build();
            mcpClientService.promptGet("http_streaming", "test", null, null, mcpDimension);
        } finally {
            mcpClientService.destroy();
            executorService.shutdown();
            EasyMock.verify(eventListenerService);
        }
    }

    @Test(expected = ExecutionException.class)
    public void testHttpStreamingWithErrorType() throws Exception {
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        mcpClientService.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        NamesService mcpNameService = ObjectBuilder.buildNamesService();
        mcpClientService.setNamesService(mcpNameService);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setTimeBetweenEvictionRunsMillis(30000);
        mcpClientService.setProcessor(10);
        mcpClientService.setTimeout(5000);
        mcpClientService.setBorrow(10000);
        mcpClientService.init(JsonUtils.read(ResourceUtils.getURL("classpath:mcp/mcp_client.json").openStream(), Map.class));
        mcpClientService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("WORLD"));
        EventListenerService eventListenerService = EasyMock.createMock(EventListenerService.class);
        eventListenerService.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListenerService);
        mcpClientService.setEventListener(eventListenerService);
        try {
            McpDimension mcpDimension = McpDimension.builder().build();
            mcpClientService.promptGet("http_streaming_error", "test", null, null, mcpDimension);
        } finally {
            mcpClientService.destroy();
            executorService.shutdown();
            EasyMock.verify(eventListenerService);
        }
    }

    @Test
    public void testBuild() throws Exception {
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EventListenerServiceImpl eventListenerManager = EasyMock.createMock(EventListenerServiceImpl.class);
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(placeholderResolver, client, eventListenerManager, executorService, notifierManager, namesService);
        Map<String, McpClientServiceImpl.McpConfig> configMap = new HashMap<>();
        McpClientServiceImpl.InitConfig mcpClientService = new McpClientServiceImpl.InitConfig();
        mcpClientService.setBorrow(100);
        mcpClientService.setHttpClient(client);
        mcpClientService.setProcessor(20);
        mcpClientService.setNotifierService(notifierManager);
        mcpClientService.setPlaceholderResolver(placeholderResolver);
        mcpClientService.setTimeout(30);
        mcpClientService.setEventListener(eventListenerManager);
        mcpClientService.setExecutorService(executorService);
        mcpClientService.setNamesService(namesService);
        McpClientServiceImpl empty = (McpClientServiceImpl) mcpClientService.mcpClientService();
        Assert.assertEquals(Integer.valueOf(100), empty.getBorrow());
        Assert.assertEquals(Integer.valueOf(20), empty.getProcessor());
        Assert.assertEquals(Integer.valueOf(30), empty.getTimeout());
        Assert.assertEquals(configMap, empty.getClients());
        Assert.assertEquals(client, empty.getHttpClient());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(placeholderResolver, empty.getPlaceholderResolver());
        Assert.assertEquals(eventListenerManager, empty.getEventListener());
        Assert.assertEquals(namesService, empty.getNamesService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        EasyMock.verify(placeholderResolver, client, eventListenerManager, executorService, notifierManager, namesService);
    }

    @Test
    public void testBuildHistoryWithInValid() {
        Map<String, Object> result1 = new HashMap<>();
        Map<String, Object> content1 = new HashMap<>();
        result1.put("content", content1);
        result1.put("role", "user");
        content1.put("type", "text");
        content1.put("text", "HELLO");
        // InValid
        Map<String, Object> result2 = new HashMap<>();
        Map<String, Object> content2 = new HashMap<>();
        result2.put("content", content2);
        content2.put("type", "text");
        content2.put("text", "HELLO");
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
        List<History> histories = mcpClientService.buildHistories(Arrays.asList(result1, result2));
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(histories.size()));
        Assert.assertEquals("HELLO", histories.getFirst().getContent());
        Assert.assertEquals(History.ROLE_USER, histories.getFirst().getRole());
    }

    @Test
    public void testPromptGet() throws Exception {
        McpClientServiceImpl mcpClientService = new McpClientServiceImpl() {
            @Override
            public McpResult<List<History>> promptGet(String client, String name, Map<String, Object> arguments, McpRuntime mcpRuntime, McpDimension mcpDimension) throws Exception {
                Assert.assertEquals("A", client);
                Assert.assertEquals("B", name);
                Assert.assertTrue(CollectionUtils.isEmpty(arguments));
                return null;
            }
        };
        mcpClientService.promptGet("A", "B", new HashMap<>(), null);
    }


    @org.junit.jupiter.api.Test
    public void testMcpClientAdditional() {
        McpClientServiceImpl service = new McpClientServiceImpl();
        org.junit.jupiter.api.Assertions.assertNotNull(service);
    }

    /** 返回在 get(timeout, unit) 时抛出 TimeoutException 的 ExecutorService，用于覆盖 catch (TimeoutException) { future.cancel(true); throw e; } */
    private static ExecutorService timeoutThrowingExecutor() {
        return new ExecutorService() {
            @Override
            public <T> Future<T> submit(Callable<T> task) {
                return new Future<T>() {
                    @Override
                    public T get(long timeout, TimeUnit unit) throws TimeoutException {
                        throw new TimeoutException();
                    }
                    @Override
                    public T get() { throw new UnsupportedOperationException(); }
                    @Override
                    public boolean cancel(boolean mayInterruptIfRunning) { return true; }
                    @Override
                    public boolean isCancelled() { return false; }
                    @Override
                    public boolean isDone() { return false; }
                };
            }
            @Override
            public <T> Future<T> submit(Runnable task, T result) { return submit(() -> result); }
            @Override
            public Future<?> submit(Runnable task) { return submit((Callable<?>) null); }
            @Override
            public void execute(Runnable command) {}
            @Override
            public void shutdown() {}
            @Override
            public List<Runnable> shutdownNow() { return Collections.emptyList(); }
            @Override
            public boolean isShutdown() { return false; }
            @Override
            public boolean isTerminated() { return false; }
            @Override
            public boolean awaitTermination(long timeout, TimeUnit unit) { return false; }
            @Override
            public <T> List<Future<T>> invokeAll(java.util.Collection<? extends Callable<T>> tasks) { throw new UnsupportedOperationException(); }
            @Override
            public <T> List<Future<T>> invokeAll(java.util.Collection<? extends Callable<T>> tasks, long timeout, TimeUnit unit) { throw new UnsupportedOperationException(); }
            @Override
            public <T> T invokeAny(java.util.Collection<? extends Callable<T>> tasks) { throw new UnsupportedOperationException(); }
            @Override
            public <T> T invokeAny(java.util.Collection<? extends Callable<T>> tasks, long timeout, TimeUnit unit) { throw new UnsupportedOperationException(); }
        };
    }

    private static McpClientServiceImpl serviceWithTimeoutExecutor(String clientName) throws Exception {
        McpClientServiceImpl service = new McpClientServiceImpl();
        service.setNamesService(ObjectBuilder.buildNamesService());
        service.setExecutorService(timeoutThrowingExecutor());
        service.setPlaceholderResolver(ObjectBuilder.buildEmptyPlaceholderResolver());
        GenericObjectPoolConfig<McpClient> poolConfig = new GenericObjectPoolConfig<>();
        poolConfig.setMaxTotal(1);
        poolConfig.setMaxIdle(1);
        GenericObjectPool<McpClient> pool = new GenericObjectPool<>(new BasePooledObjectFactory<McpClient>() {
            @Override
            public McpClient create() {
                return EasyMock.createNiceMock(McpClient.class);
            }
            @Override
            public PooledObject<McpClient> wrap(McpClient obj) {
                return new DefaultPooledObject<>(obj);
            }
        }, poolConfig);
        McpClientServiceImpl.McpConfig config = McpClientServiceImpl.McpConfig.builder()
                .client(pool)
                .timeout(100)
                .build();
        service.getClients().put(clientName, config);
        EventListenerService eventListener = EasyMock.createMock(EventListenerService.class);
        eventListener.listen(EasyMock.anyObject(Event.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(eventListener);
        service.setEventListener(eventListener);
        return service;
    }

    @Test(expected = TimeoutException.class)
    public void toolsCall_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.toolsCall("timeout-client", "toolName", Collections.emptyMap(), dim);
    }

    @Test(expected = TimeoutException.class)
    public void toolsList_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.toolsList("timeout-client", dim);
    }

    @Test(expected = TimeoutException.class)
    public void promptGet_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.promptGet("timeout-client", "promptName", Collections.emptyMap(), null, dim);
    }

    @Test(expected = TimeoutException.class)
    public void promptList_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.promptList("timeout-client", dim);
    }

    @Test(expected = TimeoutException.class)
    public void resourcesTemplatesList_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.resourcesTemplatesList("timeout-client", dim);
    }

    @Test(expected = TimeoutException.class)
    public void resourcesList_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.resourcesList("timeout-client", dim);
    }

    @Test(expected = TimeoutException.class)
    public void resourcesRead_timeoutException_cancelsFutureAndRethrows() throws Exception {
        McpClientServiceImpl service = serviceWithTimeoutExecutor("timeout-client");
        McpDimension dim = McpDimension.builder().build();
        service.resourcesRead("timeout-client", "some-uri", null, dim);
    }

}