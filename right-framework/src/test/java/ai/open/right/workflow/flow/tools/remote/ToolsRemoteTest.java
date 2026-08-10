package ai.open.right.workflow.flow.tools.remote;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.TimeoutConfig;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import ai.open.right.workflow.flow.tools.ToolsHeader;
import ai.open.right.workflow.flow.tools.ToolsOrchestrator;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.IOUtils;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.apache.http.impl.nio.client.HttpAsyncClientBuilder;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.util.Arrays;
import java.util.Map;

public class ToolsRemoteTest {

    @Test
    public void testGetToolsRequestWithEmptyWrap() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("HELLO WORLD");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        Assert.assertEquals("", remote.getToolsRequest(toolsConfig, workflowTask));
    }

    @Test
    public void testGetToolsRequestWithGetAndSourceWrap() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setWrap(ToolsConfig.WRAP_SOURCE);
        toolsConfig.setMethod("GET");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("HELLO WORLD");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        Assert.assertEquals("HELLO WORLD", remote.getToolsRequest(toolsConfig, workflowTask));
    }

    @Test(expected = WorkflowException.class)
    public void testGetToolsRequestWithBefore() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        ToolsConfig toolsConfig = new ToolsConfig();
        ToolsOrchestrator toolsFlow = new ToolsOrchestrator();
        toolsFlow.setBefore("NEXT");
        toolsConfig.setToolsOrchestrator(toolsFlow);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("HELLO WORLD");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        remote.setTimeout4Service(1000);
        remote.setTimeout4Llm(1000);
        remote.setNotifierService(notifierManager);
        remote.getToolsRequest(toolsConfig, workflowTask);
    }

    @Test
    public void testGetToolsResponse() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("HELLO WORLD");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        Assert.assertEquals("HELLO", remote.getToolsResponse(toolsConfig, ObjectBuilder.buildWorkflowTask(), new ByteArrayInputStream("HELLO".getBytes())));
    }

    @Test(expected = RuntimeException.class)
    public void testGetToolsResponseWithAfterWithException() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithWriteBackException();
        ToolsConfig toolsConfig = new ToolsConfig();
        ToolsOrchestrator toolsFlow = new ToolsOrchestrator();
        toolsFlow.setAfter("NEXT");
        toolsConfig.setToolsOrchestrator(toolsFlow);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("HELLO WORLD");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        remote.setTimeout4Service(1000);
        remote.setTimeout4Llm(1000);
        remote.setNotifierService(notifierManager);
        remote.getToolsResponse(toolsConfig, ObjectBuilder.buildWorkflowTask(), new ByteArrayInputStream("HELLO".getBytes()));
    }

    @Test
    public void testGetToolsResponseWithAfter() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildAssertNotifierManagerWithWriteBackDirect("HELLO");
        ToolsConfig toolsConfig = new ToolsConfig();
        ToolsOrchestrator toolsFlow = new ToolsOrchestrator();
        toolsFlow.setAfter("NEXT");
        toolsConfig.setToolsOrchestrator(toolsFlow);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery("HELLO WORLD");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        remote.setTimeout4Service(10000);
        remote.setTimeout4Llm(10000);
        remote.setNotifierService(notifierManager);
        Assert.assertEquals("HELLO", remote.getToolsResponse(toolsConfig, ObjectBuilder.buildWorkflowTask(), new ByteArrayInputStream("HELLO".getBytes())));
    }

    @Test
    public void testBuildRequestWithGet() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        HttpRequestBase requestBase = remote.buildRequest(toolsConfig, ObjectBuilder.buildWorkflowTask(), "HELLO_WORLD");
        Assert.assertEquals("https://www.w3.org/?HELLO_WORLD", requestBase.getURI().toString());
    }

    @Test
    public void testBuildRequestWithPost() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("Post");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        HttpRequestBase requestBase = remote.buildRequest(toolsConfig, ObjectBuilder.buildWorkflowTask(), "{\"KEY\":\"VAL\"}");
        Assert.assertEquals("{\"KEY\":\"VAL\"}", IOUtils.toString(HttpPost.class.cast(requestBase).getEntity().getContent()));
    }

    @Test
    public void testBuildRequestWithHeader() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsServiceImpl remote = new ToolsServiceImpl();
        HttpRequestBase requestBase = remote.buildRequest(toolsConfig, ObjectBuilder.buildWorkflowTask(), "HELLO_WORLD");
        Assert.assertEquals("Content-Type: application/json", requestBase.getFirstHeader("Content-Type").toString());
    }

    @Test
    public void testExecute() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsOrchestrator toolsFlow = new ToolsOrchestrator();
        toolsFlow.setParam("query");
        toolsConfig.setToolsOrchestrator(toolsFlow);
        ToolsServiceImpl remote = new ToolsServiceImpl();
        remote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect());
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        remote.setToolsClient(client);
        remote.setTimeout4Service(10000);
        remote.setTimeout4Llm(10000);
        Assert.assertTrue(remote.execute(toolsConfig, ObjectBuilder.buildWorkflowTask()).length() > 100);
        client.close();
    }

    @Test(expected = WorkflowException.class)
    public void testExecuteWithCode301() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("POST");
        TimeoutConfig timeoutConfig = new TimeoutConfig();
        timeoutConfig.setTimeout4Service(10000);
        timeoutConfig.setTimeout(10000);
        toolsConfig.setTimeout(timeoutConfig);
        ToolsOrchestrator toolsFlow = new ToolsOrchestrator();
        toolsFlow.setParam("query");
        toolsConfig.setToolsOrchestrator(toolsFlow);
        ToolsServiceImpl remote = new ToolsServiceImpl();
        CloseableHttpAsyncClient client = HttpAsyncClientBuilder.create().build();
        client.start();
        remote.setToolsClient(client);
        remote.setTimeout4Service(1000);
        remote.setTimeout4Llm(1000);
        try {
            remote.execute(toolsConfig, ObjectBuilder.buildWorkflowTask());
        } finally {
            client.close();
        }
    }

    @Test
    public void testBuildHeaders() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsHeader toolsHeader = new ToolsHeader();
        toolsHeader.setKey("Hello");
        toolsHeader.setVal("Value");
        toolsConfig.setHeaders(Arrays.asList(toolsHeader));
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        HttpRequestBase requestBase = toolsRemote.buildRequest(toolsConfig, workflowTask, "HELLO_WORLD");
        toolsRemote.buildHeaders(requestBase, toolsConfig, workflowTask);
        Assert.assertEquals("Value", requestBase.getFirstHeader("Hello").getValue());
    }

    @Test
    public void testBuildHeadersWithWorkflow() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsHeader toolsHeader = new ToolsHeader();
        toolsHeader.setKey("Hello");
        toolsHeader.setDynamic("WORKFLOW");
        toolsConfig.setHeaders(Arrays.asList(toolsHeader));
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl();
        toolsRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("Value"));
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        HttpRequestBase requestBase = toolsRemote.buildRequest(toolsConfig, workflowTask, "HELLO_WORLD");
        toolsRemote.buildHeaders(requestBase, toolsConfig, workflowTask);
        Assert.assertEquals("Value", requestBase.getFirstHeader("Hello").getValue());
    }

    @Test(expected = RuntimeException.class)
    public void testBuildHeadersWithWorkflowAndException() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsHeader toolsHeader = new ToolsHeader();
        toolsHeader.setKey("Hello");
        toolsHeader.setDynamic("WORKFLOW");
        toolsConfig.setHeaders(Arrays.asList(toolsHeader));
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl();
        toolsRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        HttpRequestBase requestBase = toolsRemote.buildRequest(toolsConfig, workflowTask, "HELLO_WORLD");
        toolsRemote.buildHeaders(requestBase, toolsConfig, workflowTask);
        Assert.assertEquals("Value", requestBase.getFirstHeader("Hello").getValue());
    }

    @Test
    public void testBuildHeadersWithWorkflowAndExceptionAndStopOnFailedFalse() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setService("https://www.w3.org/");
        toolsConfig.setMethod("GET");
        ToolsHeader toolsHeader = new ToolsHeader();
        toolsHeader.setDynamic("WORKFLOW");
        toolsHeader.setKey("Hello");
        toolsHeader.setVal("Value");
        toolsHeader.setStopOnFailed(false);
        toolsConfig.setHeaders(Arrays.asList(toolsHeader));
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl();
        toolsRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackException());
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        HttpRequestBase requestBase = toolsRemote.buildRequest(toolsConfig, workflowTask, "HELLO_WORLD");
        toolsRemote.buildHeaders(requestBase, toolsConfig, workflowTask);
        Assert.assertEquals("Value", requestBase.getFirstHeader("Hello").getValue());
    }

    @Test
    public void testWrapWithObject() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setWrap(ToolsConfig.WRAP_OBJECT);
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl();
        String response = toolsRemote.wrapRequest(toolsConfig, ObjectBuilder.buildWorkflowTaskWithTimestamp(100L), "{\"KEY\":\"VAL\",\"KEY2\":{\"KEY3\":\"VAL3\"}}");
        Assert.assertTrue(JsonUtils.read(response, Map.class) != null);
    }

    @Test
    public void testWrapWithString() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setWrap(ToolsConfig.WRAP_STRING);
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl();
        String response = toolsRemote.wrapRequest(toolsConfig, ObjectBuilder.buildWorkflowTaskWithTimestamp(100L), "{\"KEY\":\"VAL\",\"KEY2\":{\"KEY3\":\"VAL3\"}}");
        Assert.assertTrue(JsonUtils.read(response, Map.class) != null);
    }

    @Test
    public void testGetToolsRequestWithOrchestratorBefore() throws Exception {
        ToolsConfig toolsConfig = new ToolsConfig();
        ToolsOrchestrator toolsOrchestrator = new ToolsOrchestrator();
        toolsOrchestrator.setBefore("BEFORE");
        toolsConfig.setToolsOrchestrator(toolsOrchestrator);
        ToolsServiceImpl toolsRemote = new ToolsServiceImpl() {
            public String wrapRequest(ToolsConfig toolsConfig, WorkflowTask workTask, String response) throws Exception {
                return "HELLO";
            }
        };
        toolsRemote.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect());
        Assert.assertEquals("HELLO", toolsRemote.getToolsRequest(toolsConfig, ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testInit() throws Exception {
        CloseableHttpAsyncClient client = EasyMock.createMock(CloseableHttpAsyncClient.class);
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        EasyMock.replay(client);
        ToolsServiceImpl.InitConfig service = new ToolsServiceImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setToolsClient(client);
        service.setTimeout4Llm(1000);
        service.setTimeout4Service(2000);
        ToolsServiceImpl empty = (ToolsServiceImpl) service.toolsService();
        Assert.assertEquals(client, empty.getToolsClient());
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
        Assert.assertEquals(Integer.valueOf(2000), empty.getTimeout4Service());
        EasyMock.verify(client);
    }
}
