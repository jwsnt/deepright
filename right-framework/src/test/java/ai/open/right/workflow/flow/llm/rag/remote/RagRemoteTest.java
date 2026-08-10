package ai.open.right.workflow.flow.llm.rag.remote;
import org.apache.http.client.methods.HttpGet;
import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagOrchestrator;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.StatusLine;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.util.Arrays;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

public class RagRemoteTest {

    @Test
    public void testHashCode1() throws Exception {
        Object object = RagRemote.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = RagRemote.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testBuildRequestPost() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        Assert.assertEquals(httpRequestBase.getMethod(), "POST");
    }

    @Test
    public void testBuildRequestGet() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        Assert.assertEquals(httpRequestBase.getMethod(), "GET");
    }

    @Test
    public void testGetHttpResponse() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        Future future = EasyMock.createMock(Future.class);
        EasyMock.expect(future.get(1000, TimeUnit.MILLISECONDS)).andReturn(null).anyTimes();
        CloseableHttpAsyncClient ragClient = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.expect(ragClient.execute(httpRequestBase, null)).andReturn(future).anyTimes();
        EasyMock.replay(ragClient, future);
        ragRemote.setRagClient(ragClient);
        ragRemote.setTimeout4Service(1000);
        HttpResponse response = ragRemote.getResponse(ragConfig, httpRequestBase);
        Assert.assertNull(response);
        EasyMock.verify(ragClient, future);
    }

    @Test
    public void testGetRagRequest() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagRemote ragRemote = new RagRemote();
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        Assert.assertEquals(future.getRequest().length(), 331);
    }

    @Test
    public void testGetRagRequestWithEmpty() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagRemote ragRemote = new RagRemote();
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        Assert.assertEquals(future.getRequest().length(), 0);
    }


    @Test(expected = RuntimeException.class)
    public void testGetRagRequestHasBeforeWithException() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagOrchestrator ragFlow = new RagOrchestrator();
        ragFlow.setBefore("NEXT CHAIN");
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setRagOrchestrator(ragFlow);
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackException();
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(notifierManager);
        ragRemote.setTimeout4Llm(1000);
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        future.getRequest();
    }

    @Test
    public void testGetRagRequestHasBefore() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagOrchestrator ragFlow = new RagOrchestrator();
        ragFlow.setBefore("NEXT CHAIN");
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setRagOrchestrator(ragFlow);
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(notifierManager);
        ragRemote.setTimeout4Llm(1000);
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        Assert.assertEquals("HELLO", future.getRequest());
    }

    @Test
    public void testGetRagResponse() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagRemote ragRemote = new RagRemote();
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        future.getResponse(new ByteArrayInputStream(new byte[]{1, 2, 3}));
    }

    @Test
    public void testGetRagResponseWithAfter() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagOrchestrator ragFlow = new RagOrchestrator();
        ragFlow.setAfter("NEXT CHAIN");
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setRagOrchestrator(ragFlow);
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("HELLO");
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(notifierManager);
        ragRemote.setTimeout4Llm(1000);
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        Assert.assertEquals("HELLO", future.getResponse(new ByteArrayInputStream(new byte[]{1, 2, 3})));
    }

    @Test(expected = RuntimeException.class)
    public void testGetRagResponseWithAfterAndException() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagOrchestrator ragFlow = new RagOrchestrator();
        ragFlow.setAfter("NEXT CHAIN");
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setRagOrchestrator(ragFlow);
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackException();
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(notifierManager);
        ragRemote.setTimeout4Llm(1000);
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData);
        future.getResponse(new ByteArrayInputStream(new byte[]{1, 2, 3}));
    }

    @Test
    public void testCall() throws Exception {
        RagConfig ragConfig = new RagConfig();
        RagOrchestrator ragFlow = new RagOrchestrator();
        ragFlow.setAfter("NEXT CHAIN");
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setRagOrchestrator(ragFlow);
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        HttpResponse response = EasyMock.createMock(HttpResponse.class);
        StatusLine statusLine = EasyMock.createMock(StatusLine.class);
        EasyMock.expect(response.getStatusLine()).andReturn(statusLine).anyTimes();
        EasyMock.expect(statusLine.getStatusCode()).andReturn(200).anyTimes();
        HttpEntity entity = EasyMock.createMock(HttpEntity.class);
        EasyMock.expect(response.getEntity()).andReturn(entity).anyTimes();
        EasyMock.expect(entity.getContent()).andReturn(new ByteArrayInputStream(new byte[]{})).anyTimes();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        EasyMock.replay(response, statusLine, entity);
        RagRemote ragRemote = new RagRemote() {
            protected HttpResponse getResponse(RagConfig ragConfig, HttpRequestBase httpRequest) throws Exception {
                return response;
            }
        };
        ragRemote.setNotifierService(notifierManager);
        ragRemote.setTimeout4Service(1000);
        ragRemote.setTimeout4Llm(1000);
        ragRemote.setTimeout(1000);
        RagRemote.RemoteFuture future = ragRemote.new RemoteFuture(ragConfig, ragData) {
            protected String getResponse(InputStream input) throws Exception {
                return "HELLO";
            }
        };
        future.call();
        EasyMock.verify(response, statusLine, entity);
    }

    @Test
    public void testRag() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("POST");
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        RagRemote ragRemote = new RagRemote();
        ragRemote.setExecutorService(executorService);
        ragRemote.rag(ragConfig, ragData);
        executorService.shutdown();
    }

    @Test
    public void testWithConditionFailed() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false");
        ExecutorService executorService = Executors.newFixedThreadPool(1);
        RagData ragData = RagData.builder()
                .query(ObjectBuilder.buildLLMQuery())
                .config(new LLMConfig())
                .prompt("HELLO")
                .build();
        RagRemote remote = new RagRemote();
        remote.setNotifierService(notifierManager);
        remote.setExecutorService(executorService);
        RagConfig ragConfig = new RagConfig();
        ragConfig.setCondition("Workflow2");
        Assert.assertEquals(RagFuture.NOTHING, remote.rag(ragConfig, ragData));
    }

    @Test
    public void testBuildHeaders() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        RagRemoteHeader ragRemoteHeader = new RagRemoteHeader();
        ragRemoteHeader.setKey("Hello");
        ragRemoteHeader.setVal("Value");
        ragRemoteConfig.setHeaders(Arrays.asList(ragRemoteHeader));
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        ragRemote.buildHeaders(httpRequestBase, ragConfig, ragData);
        Assert.assertEquals("Value", httpRequestBase.getFirstHeader("Hello").getValue());
    }

    @Test
    public void testBuildHeadersWithWorkflow() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        RagRemoteHeader ragRemoteHeader = new RagRemoteHeader();
        ragRemoteHeader.setKey("Hello");
        ragRemoteHeader.setDynamic("Value");
        ragRemoteConfig.setHeaders(Arrays.asList(ragRemoteHeader));
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("Value"));
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        ragRemote.buildHeaders(httpRequestBase, ragConfig, ragData);
        Assert.assertEquals("Value", httpRequestBase.getFirstHeader("Hello").getValue());
    }

    @Test(expected = RuntimeException.class)
    public void testBuildHeadersWithWorkflowAndException() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        RagRemoteHeader ragRemoteHeader = new RagRemoteHeader();
        ragRemoteHeader.setKey("Hello");
        ragRemoteHeader.setDynamic("Value");
        ragRemoteConfig.setHeaders(Arrays.asList(ragRemoteHeader));
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        ragRemote.buildHeaders(httpRequestBase, ragConfig, ragData);
    }

    @Test
    public void testBuildHeadersWithWorkflowAndExceptionAndStopOnFailedFalse() throws Exception {
        RagConfig ragConfig = new RagConfig();
        ragConfig.setService("https://www.w3.org/");
        ragConfig.setReplace("#key");
        ragConfig.setMethod("GET");
        RagRemoteConfig ragRemoteConfig = new RagRemoteConfig();
        RagRemoteHeader ragRemoteHeader = new RagRemoteHeader();
        ragRemoteHeader.setKey("Hello");
        ragRemoteHeader.setVal("Olleh");
        ragRemoteHeader.setDynamic("Value");
        ragRemoteHeader.setStopOnFailed(false);
        ragRemoteConfig.setHeaders(Arrays.asList(ragRemoteHeader));
        ragConfig.setRagRemoteConfig(ragRemoteConfig);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(llmConfig)
                .query(llmQuery)
                .build();
        RagRemote ragRemote = new RagRemote();
        ragRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        HttpRequestBase httpRequestBase = ragRemote.buildRequest(ragConfig, ragData, "Hello");
        ragRemote.buildHeaders(httpRequestBase, ragConfig, ragData);
        Assert.assertEquals("Olleh", httpRequestBase.getFirstHeader("Hello").getValue());
    }

    @Test
    public void testInit() throws Exception {
        ExecutorService executorService = EasyMock.createMock(ExecutorService.class);
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.replay(client, executorService);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        RagRemote.InitConfig service = new RagRemote.InitConfig();
        service.setNotifierService(notifierManager);
        service.setExecutorService(executorService);
        service.setRagClient(client);
        service.setTimeout(1000);
        service.setTimeout4Llm(2000);
        service.setTimeout4Service(3000);
        service.setTimeout4Condition(10086);
        RagRemote empty = service.ragRemote();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(executorService, empty.getExecutorService());
        Assert.assertEquals(client, empty.getRagClient());
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        Assert.assertEquals(Integer.valueOf(2000), empty.getTimeout4Llm());
        Assert.assertEquals(Integer.valueOf(3000), empty.getTimeout4Service());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout());
        EasyMock.verify(client, executorService);
    }
    @Test(expected = IllegalArgumentException.class)
    public void testBuildHeadersEmptyKey() throws Exception {
        RagRemote remote = new RagRemote();
        RagConfig config = new RagConfig();
        RagRemoteConfig rrc = new RagRemoteConfig();
        RagRemoteHeader header = new RagRemoteHeader();
        header.setKey("");
        rrc.setHeaders(Arrays.asList(header));
        config.setRagRemoteConfig(rrc);
        remote.buildHeaders(new HttpGet("http://x"), config, null);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testBuildRequestEmptyService() throws Exception {
        RagRemote remote = new RagRemote();
        RagConfig config = new RagConfig();
        config.setService("");
        remote.buildRequest(config, null, "body");
    }
}
