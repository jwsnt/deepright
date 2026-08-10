package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.config.WorkflowConfigService;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.*;
import ai.open.right.workflow.flow.llm.provider.google.GoogleRequest;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.mcp.client.McpResult;
import ai.open.right.workflow.mcp.client.McpRuntime;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.dimension.McpDimensionService;
import ai.open.right.workflow.mcp.client.impl.McpClientServiceImpl;
import ai.open.right.workflow.mcp.client.rewrtier.impl.McpRewriteServiceImpl;
import ai.open.right.workflow.mcp.client.trigger.impl.McpTriggerServiceImpl;
import com.google.common.collect.ImmutableMap;
import org.easymock.Capture;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.CollectionUtils;
import org.springframework.util.ResourceUtils;

import java.util.*;

public class ProviderRequestServiceTest {

    /**
     * History 默认 created 为当前毫秒；internalHistory 会剔除 created &gt;= message.timestamp 的条目，测试中需显式设为早于消息时间
     */
    private static void setHistoryStrictlyBeforeMessage(Message message, History... histories) {
        long ts = message.getCreated();
        for (History h : histories) {
            h.setCreated(ts - 1L);
        }
    }

    @Test
    public void testRequest() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.hasRecallOffset()).andReturn(false).anyTimes();
        EasyMock.expect(request.getRecallFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(10).anyTimes();
        message.setWorkflow("WORKFLOW");
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        List<History> histories = new ArrayList<>();
        histories.add(new History());
        EasyMock.replay(request);
        EasyMock.expect(historyStore.restore(message, "WORKFLOW", 10, false, -request.getMessage().getCreated(), 0L)).andReturn(histories).anyTimes();
        EasyMock.replay(historyStore);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        providerRequestService.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        providerRequestService.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        providerRequestService.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertNotNull(providerRequestService.getProviderRequestRewriter());
        EasyMock.verify(request, historyStore);
    }

    /**
     * internalHistory：recallFunCall=false 时在「恢复结果」上剔除 FUN_FUNCALL；最终列表为 replaceHistories，不与原 message 端上记忆合并
     */
    @Test
    public void testInternalHistory_recallFunCallFalse_filtersFunctionHistories() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        request.setMessage(message);
        request.setScene("WORKFLOW");
        request.setClientHistories(false);
        request.setContainHistories(true);
        request.setRecallFunCall(false);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        List<History> restored = new ArrayList<>();
        History restoredFunc = new History();
        restoredFunc.setFunction(History.FUN_FUNCALL);
        restoredFunc.setContent("restored-func");
        History restoredChat = new History();
        restoredChat.setFunction(History.FUN_CHAT);
        restoredChat.setContent("restored-chat");
        restored.add(restoredFunc);
        restored.add(restoredChat);
        setHistoryStrictlyBeforeMessage(message, restoredFunc, restoredChat);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("WORKFLOW"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong())).andReturn(restored).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> result = request.getMessage().getHistories();
        Assert.assertNotNull(result);
        Assert.assertEquals(1, result.size());
        Assert.assertEquals("restored-chat", result.get(0).getContent());
        Assert.assertFalse(result.stream().anyMatch(h -> h.isFunction(History.FUN_FUNCALL)));
        EasyMock.verify(historyStore);
    }

    @Test
    public void testRequestWithCustomerTools() throws Exception {
        Map<String, Object> metadata = JsonUtils.read(ResourceUtils.getURL("classpath:CustomerTools.json").openStream(), Map.class);
        Message message = EasyMock.createMock(Message.class);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        request.setFunCalls(EasyMock.anyObject());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(message.getMetadata(ProviderRequestService.KEY_FUN_INTERNAL, List.class)).andReturn(List.class.cast(metadata.get(ProviderRequestService.KEY_FUN_INTERNAL))).anyTimes();
        EasyMock.replay(request, message);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.buildFunCall(request, llmConfig);
        EasyMock.verify(request, message);
    }

    @Test
    public void testRequestWithDecoration() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WORKFLOW");
        request.setMessage(message);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            protected void buildPrompt(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setProviderToken(new ProviderToken());
        providerRequestService.request(request, llmConfig, message);
    }

    @Test
    public void testRequestWithNotifier() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        message.setNotifier("NOTIFIER");
        request.setMessage(message);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            protected void buildPrompt(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setNotifier("NONE");
        providerRequestService.setProviderToken(new ProviderToken());
        providerRequestService.request(request, llmConfig, message);
        Assert.assertEquals("NONE", request.getNotifier());
    }

    @Test
    public void testFunCallsWithOriginal() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        EasyMock.expect(workflowConfigService.config("UNKNOWN", "HELLO")).andReturn(workflowConfig);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("HELLO");
        funCalls.add(llmFunCall);
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        EasyMock.replay(workflowConfigService);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.expect(namesService.encode("Workflow_", "UNKNOWN", "HELLO")).andReturn("__").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "UNKNOWN", "__")).andReturn("__").anyTimes();
        EasyMock.replay(namesService);
        providerRequestService.setNamesService(namesService);
        providerRequestService.recallFunCall(request, new LLMConfig(), funCalls);
        Assert.assertEquals("__", request.getFunCalls().getFirst().getName());
        Assert.assertEquals(workflowConfigService, providerRequestService.getWorkflowConfigService());
        EasyMock.verify(workflowConfigService, namesService);
    }

    @Test
    public void testWorkflowFunCallsException() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("HELLO");
        funCalls.add(llmFunCall);
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        // Null Point
        providerRequestService.recallFunCall(request, new LLMConfig(), funCalls);
        Assert.assertTrue(CollectionUtils.isEmpty(request.getFunCalls()));
    }

    @Test
    public void testFunCalls() throws Exception {
        McpDimension mcpDimension = McpDimension.builder().build();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return mcpDimension;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = ObjectBuilder.buildNamesService();
        String encode1 = namesService.encode(NamesService.PREFIX_TOOLS, "CLIENT", "HELLO");
        String encode2 = namesService.encode(NamesService.PREFIX_TOOLS, "CLIENT", "WROLD");
        providerRequestService.setNamesService(namesService);
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName(encode1);
        llmFunCall.setRefer(true);
        funCalls.add(llmFunCall);
        List<ProviderFunCall> providerFunCalls = new ArrayList<ProviderFunCall>();
        LLMFunCall providerFunCall = new LLMFunCall();
        providerFunCall.setName(encode2);
        providerFunCalls.add(providerFunCall);
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        EasyMock.expect(mcpClientService.toolsList(encode1, mcpDimension)).andReturn(providerFunCalls).anyTimes();
        EasyMock.replay(mcpClientService);
        providerRequestService.setMcpClientService(mcpClientService);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setFunCalls(funCalls);
        providerRequestService.buildFunCall(request, llmConfig);
        Assert.assertEquals(encode2, request.getFunCalls().getFirst().getName());
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testFunCallsWithWhiteList() throws Exception {
        McpDimension mcpDimension = McpDimension.builder().build();
        ProviderRequestService providerRequestService = new ProviderRequestService<ProviderRequest>() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return mcpDimension;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = ObjectBuilder.buildNamesService();
        providerRequestService.setNamesService(namesService);
        String encode1 = namesService.encode(NamesService.PREFIX_TOOLS, "CLIENT", "HELLO");
        String encode2 = namesService.encode(NamesService.PREFIX_TOOLS, "CLIENT", "WORLD");
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setWhiteList(Arrays.asList("WORLD"));
        llmFunCall.setName(encode1);
        llmFunCall.setRefer(true);
        funCalls.add(llmFunCall);
        List<ProviderFunCall> providerFunCalls = new ArrayList<ProviderFunCall>();
        LLMFunCall providerFunCall = new LLMFunCall();
        providerFunCall.setName(encode1);
        providerFunCalls.add(providerFunCall);
        providerFunCall = new LLMFunCall();
        providerFunCall.setName(encode2);
        providerFunCalls.add(providerFunCall);
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        EasyMock.expect(mcpClientService.toolsList(encode1, mcpDimension)).andReturn(providerFunCalls).anyTimes();
        EasyMock.replay(mcpClientService);
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setFunCalls(funCalls);
        providerRequestService.buildFunCall(request, llmConfig);
        Assert.assertEquals(encode2, request.getFunCalls().getFirst().getName());
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testFunCallsWithCustomer() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.getMessage().getMetadata().put(ProviderRequestService.KEY_FUN_INTERNAL, JsonUtils.read("[{\n" + "                        \"description\": \"获取天气\",\n" + "                        \"name\": \"workflow2\",\n" + "                        \"properties\": {\n" + "                            \"city\": {\n" + "                                \"type\": \"string\",\n" + "                                \"description\": \"城市名称,可以是多个\"\n" + "                            },\n" + "                            \"description\": {\n" + "                                \"type\": \"string\",\n" + "                                \"description\": \"时间范围，比如最近几个月\"\n" + "                            }\n" + "                        },\n" + "                        \"required\": [\n" + "                            \"city\"\n" + "                        ]\n" + "                    },\n" + "                    {\n" + "                        \"description\": \"获取用户所在地区\",\n" + "                        \"name\": \"workflow3\",\n" + "                        \"properties\": {\n" + "                            \"location\": {\n" + "                                \"type\": \"string\",\n" + "                                \"description\": \"比如华东或华北\"\n" + "                            }\n" + "                        },\n" + "                        \"required\": [\n" + "                            \"location\"\n" + "                        ]\n" + "                    }\n" + "                ]", List.class));
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(ObjectBuilder.buildNamesService());
        providerRequestService.buildFunCall(request, llmConfig);
        Assert.assertEquals("workflow2", request.getFunCalls().getFirst().getName());
    }

    @Test
    public void testHistoryWithEmtpy() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected void internalHistory(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        providerRequestService.externalHistory(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        EasyMock.verify(request);
    }

    @Test
    public void testHistoryWithMcp() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpDimension mcpDimension = McpDimension.builder().build();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                mcpDimension.bind(new String[]{"A", "B"});
                return mcpDimension;
            }

            @Override
            protected McpRuntime buildMcpRuntime(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return mcpRuntime;
            }

            @Override
            protected void internalHistory(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setClient("CLIENT");
        List<History> history = new ArrayList<>();
        llmConfig.setMcpCall(mcpCall);
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getUserContext()).andReturn(ObjectBuilder.buildEmpty()).anyTimes();
        EasyMock.expect(message.getWorkflow()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(message.getChat()).andReturn("CHAT").anyTimes();
        EasyMock.expect(message.getBiz()).andReturn("Biz").anyTimes();
        message.addHistories(history);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpResult<List<History>> result = new McpResult<List<History>>();
        result.setResult(history);
        EasyMock.expect(mcpClientService.promptGet("A", "B", mcpCall.arguments("UNKNOWN"), mcpRuntime, mcpDimension)).andReturn(result).anyTimes();
        EasyMock.replay(request, message, mcpClientService);
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());
        providerRequestService.setMcpTriggerService(new McpTriggerServiceImpl());
        providerRequestService.externalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        EasyMock.verify(request, message, mcpClientService);
    }

    @Test
    public void testHistoryWithMcpAndReplace() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpDimension mcpDimension = McpDimension.builder().build();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                mcpDimension.bind(new String[]{"A", "B"});
                return mcpDimension;
            }

            @Override
            protected McpRuntime buildMcpRuntime(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return mcpRuntime;
            }

            @Override
            protected void internalHistory(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setClient("CLIENT");
        mcpCall.setReplace(true);
        History history = new History();
        history.setRole(History.ROLE_USER);
        history.setContent("HELLO");
        llmConfig.setMcpCall(mcpCall);
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getUserContext()).andReturn(UserContext.builder().build()).anyTimes();
        EasyMock.expect(message.getWorkflow()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(message.getChat()).andReturn("CHAT").anyTimes();
        EasyMock.expect(message.getBiz()).andReturn("Biz").anyTimes();
        message.setQuery("HELLO");
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpResult<List<History>> result = new McpResult<List<History>>();
        result.setResult(Arrays.asList(history));
        EasyMock.expect(mcpClientService.promptGet("A", "B", mcpCall.arguments("UNKNOWN"), mcpRuntime, mcpDimension)).andReturn(result).anyTimes();
        EasyMock.replay(request, message, mcpClientService);
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());
        providerRequestService.setMcpTriggerService(new McpTriggerServiceImpl());
        providerRequestService.externalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        EasyMock.verify(request, message, mcpClientService);
    }

    @Test
    public void testMcpRuntime() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(request);
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        llmMcpCall.setDynamic("DYNAMIC");
        llmMcpCall.setClient("CLIENT");
        llmMcpCall.setTimeout(1000);
        McpRuntime mcpRuntime = providerRequestService.buildMcpRuntime(request, new LLMConfig(), llmMcpCall);
        Assert.assertEquals(Integer.valueOf(1000), mcpRuntime.getTimeout());
        Assert.assertEquals("DYNAMIC", mcpRuntime.getDynamic());
        Assert.assertEquals("UNKNOWN", mcpRuntime.getWorkTask().getQuery());
        EasyMock.verify(request);
    }

    @Test
    public void testHistoryWithMcpAndReplaceFailed() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        McpRuntime mcpRuntime = McpRuntime.builder().build();
        McpDimension mcpDimension = McpDimension.builder().build();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, McpDimension mcpDimension) throws Exception {
                return mcpDimension;
            }


            @Override
            protected McpRuntime buildMcpRuntime(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return mcpRuntime;
            }

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                mcpDimension.bind(new String[]{"A", "B"});
                return mcpDimension;
            }

            @Override
            protected void internalHistory(ProviderRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {

            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setClient("CLIENT");
        mcpCall.setReplace(true);
        List<History> histories = new ArrayList<>();
        History history = new History();
        history.setRole(History.ROLE_ASSISTANT);
        history.setContent("HELLO");
        histories.add(history);
        llmConfig.setMcpCall(mcpCall);
        Message message = EasyMock.createMock(Message.class);
        EasyMock.expect(message.getUserContext()).andReturn(ObjectBuilder.buildEmpty()).anyTimes();
        EasyMock.expect(message.getWorkflow()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(message.getQuery()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(message.getChat()).andReturn("CHAT").anyTimes();
        EasyMock.expect(message.getBiz()).andReturn("Biz").anyTimes();
        message.setQuery("HELLO");
        EasyMock.expectLastCall().anyTimes();
        message.addHistories(histories);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        McpResult<List<History>> result = new McpResult<List<History>>();
        result.setResult(histories);
        EasyMock.expect(mcpClientService.promptGet("A", "B", mcpCall.arguments("UNKNOWN"), mcpRuntime, mcpDimension)).andReturn(result).anyTimes();
        EasyMock.replay(request, message, mcpClientService);
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setMcpTriggerService(new McpTriggerServiceImpl());
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());
        providerRequestService.externalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        EasyMock.verify(request, message, mcpClientService);
    }

    @Test
    public void testFunCallsWithNotMcpCall() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        llmConfig.setMcpCall(llmMcpCall);
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall funCall = new LLMFunCall();
        funCall.setName(NamesService.PREFIX_RESOURCE + "OK");
        funCalls.add(funCall);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setNamesService(ObjectBuilder.buildNamesService());
        OpenAiRequest openAiRequest = new OpenAiRequest();
        providerRequestService.addInternalFunCall(openAiRequest, new LLMConfig(), funCalls);
        Assert.assertEquals(1, openAiRequest.getFunCalls().size());
    }

    @Test
    public void testFunCallsWithMcpCall() throws Exception {
        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall llmMcpCall = new LLMMcpCall();
        llmConfig.setMcpCall(llmMcpCall);
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall funCall = new LLMFunCall();
        funCall.setName(NamesService.PREFIX_RESOURCE + "OK");
        funCall.setRefer(true);
        funCalls.add(funCall);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setNamesService(ObjectBuilder.buildNamesService());
        OpenAiRequest openAiRequest = new OpenAiRequest();
        providerRequestService.addInternalFunCall(openAiRequest, new LLMConfig(), funCalls);
        Assert.assertNull(openAiRequest.getFunCalls());
    }

    @Test
    public void testHistoryWithServiceOnAndCustomerOn() throws Exception {
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.hasRecallOffset()).andReturn(false).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getRecallFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getScene()).andReturn("ABC").anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(1).anyTimes();
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        History history1 = new History();
        history1.setContent("C1");
        history1.setCreated(1L);
        request.getMessage().addHistory(history1);
        History history2 = new History();
        history2.setContent("C2");
        history2.setCreated(1L);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(store.restore(request.getMessage(), "ABC", 1, false, -request.getMessage().getCreated(), 0L)).andReturn(List.of(history2, history1)).anyTimes();
        EasyMock.replay(store);
        providerRequestService.setHistoryStore(store);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(1);
        providerRequestService.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(request.getMessage().getHistories().size()));
        Assert.assertEquals(history2, request.getMessage().getHistories().getFirst());
        Assert.assertEquals(history1, request.getMessage().getHistories().getLast());
        EasyMock.verify(request, store);
    }

    @Test
    public void testHistoryWithServiceOffAndCustomerOn() throws Exception {
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(false).anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getScene()).andReturn("ABC").anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(1).anyTimes();
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        History history1 = new History();
        history1.setContent("C1");
        request.getMessage().addHistory(history1);
        History history2 = new History();
        history2.setContent("C2");
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(store.restore(request.getMessage(), "ABC", 1)).andReturn(List.of(history2)).anyTimes();
        EasyMock.replay(store);
        providerRequestService.setHistoryStore(store);
        providerRequestService.internalHistory(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(request.getMessage().getHistories().size()));
        Assert.assertEquals(history1, request.getMessage().getHistories().getFirst());
        EasyMock.verify(request, store);
    }

    @Test
    public void testHistoryWithServiceOnAndCustomerOff() throws Exception {
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getRecallOffset()).andReturn(100000).anyTimes();
        EasyMock.expect(request.hasRecallOffset()).andReturn(true).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getRecallFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(false).anyTimes();
        EasyMock.expect(request.getScene()).andReturn("ABC").anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(1).anyTimes();
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        History history1 = new History();
        history1.setContent("C1");
        history1.setCreated(1L);
        request.getMessage().addHistory(history1);
        History history2 = new History();
        history2.setContent("C2");
        history2.setCreated(1L);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(store.restore(request.getMessage(), "ABC", 10, false, -request.getMessage().getCreated(), -(request.getMessage().getCreated() - 100000 * 1000))).andReturn(List.of(history2)).anyTimes();
        EasyMock.replay(store);
        providerRequestService.setHistoryStore(store);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        providerRequestService.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(request.getMessage().getHistories().size()));
        Assert.assertEquals(history2, request.getMessage().getHistories().getFirst());
        EasyMock.verify(request, store);
    }

    @Test
    public void testHistoryWithServiceOnAndCustomerOff2() throws Exception {
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.hasRecallOffset()).andReturn(false).anyTimes();
        EasyMock.expect(request.getRecallFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(false).anyTimes();
        EasyMock.expect(request.getScene()).andReturn("ABC").anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(1).anyTimes();
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        History history1 = new History();
        history1.setContent("C1");
        history1.setCreated(1L);
        request.getMessage().addHistory(history1);
        History history2 = new History();
        history2.setContent("C2");
        history2.setCreated(1L);
        HistoryStore store = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(store.restore(request.getMessage(), "ABC", 15, false, -request.getMessage().getCreated(), 0L)).andReturn(List.of(history2)).anyTimes();
        EasyMock.replay(store);
        providerRequestService.setHistoryStore(store);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRecallNums(15);
        llmConfig.setHistories(10);
        providerRequestService.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(request.getMessage().getHistories().size()));
        Assert.assertEquals(history2, request.getMessage().getHistories().getFirst());
        EasyMock.verify(request, store);
    }


    @Test
    public void testHistoryWithCustomer2() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(false).anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(true).anyTimes();
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        History history = new History();
        history.setContent("HISTORY");
        request.getMessage().addHistory(history);
        providerRequestService.internalHistory(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(request.getMessage().getHistories().size()));
        Assert.assertEquals("HISTORY", request.getMessage().getHistories().getFirst().getContent());
        EasyMock.verify(request);
    }

    @Test
    public void testHistoryWithOutCustomer3() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(request.getClientDowngrade()).andReturn(false).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(false).anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(false).anyTimes();
        EasyMock.replay(request);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        History history = new History();
        history.setContent("HISTORY");
        request.getMessage().addHistory(history);
        providerRequestService.internalHistory(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Integer.valueOf(0), Integer.valueOf(request.getMessage().getHistories().size()));
        EasyMock.verify(request);
    }

    @Test
    public void testBridge() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setBridged(true);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.putMetadata("HELLO", "WORLD");
        Map<String, Object> metadata = providerRequestService.buildMetadata(null, llmConfig, llmQuery);
        Assert.assertEquals(llmQuery.getUserContext(), metadata.get("userContext"));
        Assert.assertEquals(llmQuery.getChat(), metadata.get("chat"));
        Assert.assertEquals("WORLD", metadata.get("HELLO"));
    }

    @Test
    public void testBridgeWithNull() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setBridged(false);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.putMetadata("HELLO", "WORLD");
        Map<String, Object> metadata = providerRequestService.buildMetadata(null, llmConfig, llmQuery);
        Assert.assertEquals(metadata, llmQuery.getMetadata());
    }

    @Test
    public void testWithDesc() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        NamesService namesService = ObjectBuilder.buildNamesService();
        List<ProviderFunCall> funCalls = new ArrayList<>();
        ProviderFunCall funCall = new LLMFunCall();
        funCall.setName(namesService.encode(NamesService.PREFIX_TOOLS, "A", "NAME"));
        funCall.setDescription("D");
        funCalls.add(funCall);
        EasyMock.expect(mcpClientService.toolsList(EasyMock.anyString(), EasyMock.anyObject(McpDimension.class))).andReturn(funCalls).anyTimes();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, McpDimension mcpDimension) throws Exception {
                return mcpDimension;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setNamesService(namesService);
        providerRequestService.setMcpClientService(mcpClientService);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setDescriptions(Collections.singletonMap("NAME", "WORLD"));
        LLMFunCall provider = new LLMFunCall();
        provider.setDescription("HELLO");
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(mcpClientService, request);
        Assert.assertEquals("WORLD", LLMFunCall.class.cast(providerRequestService.recallMcpFunCalls(request, new LLMConfig(), llmFunCall).getFirst()).getDescription());
        EasyMock.verify(mcpClientService, request);
    }

    @Test
    public void testWithPrefix() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        NamesService namesService = ObjectBuilder.buildNamesService();
        List<ProviderFunCall> funCalls = new ArrayList<>();
        ProviderFunCall funCall = new LLMFunCall();
        funCall.setName(namesService.encode(NamesService.PREFIX_TOOLS, "A", "NAME"));
        funCall.setDescription("HELLO");
        funCalls.add(funCall);
        EasyMock.expect(mcpClientService.toolsList(EasyMock.anyString(), EasyMock.anyObject(McpDimension.class))).andReturn(funCalls).anyTimes();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, McpDimension mcpDimension) throws Exception {
                return mcpDimension;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setNamesService(namesService);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setPrefix("P");
        llmFunCall.setSuffix("S");
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(mcpClientService, providerRequest);
        Assert.assertEquals("PHELLOS", LLMFunCall.class.cast(providerRequestService.recallMcpFunCalls(providerRequest, new LLMConfig(), llmFunCall).getFirst()).getDescription());
        EasyMock.verify(mcpClientService, providerRequest);
    }

    @Test
    public void testWithDescWithPrefix() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        NamesService namesService = ObjectBuilder.buildNamesService();
        List<ProviderFunCall> funCalls = new ArrayList<>();
        ProviderFunCall funCall = new LLMFunCall();
        funCall.setName(namesService.encode(NamesService.PREFIX_TOOLS, "A", "NAME"));
        funCall.setDescription("D");
        funCalls.add(funCall);
        EasyMock.expect(mcpClientService.toolsList(EasyMock.anyString(), EasyMock.anyObject(McpDimension.class))).andReturn(funCalls).anyTimes();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, McpDimension mcpDimension) throws Exception {
                return mcpDimension;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setNamesService(namesService);
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setDescriptions(Collections.singletonMap("NAME", "WORLD"));
        llmFunCall.setPrefix("P");
        llmFunCall.setSuffix("S");
        LLMFunCall provider = new LLMFunCall();
        provider.setDescription("HELLO");
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(mcpClientService, providerRequest);
        providerRequestService.recallMcpFunCalls(providerRequest, new LLMConfig(), llmFunCall);
        Assert.assertEquals("PWORLDS", LLMFunCall.class.cast(providerRequestService.recallMcpFunCalls(providerRequest, new LLMConfig(), llmFunCall).getFirst()).getDescription());
        EasyMock.verify(mcpClientService, providerRequest);
    }

    @Test
    public void testWithProperties() throws Exception {
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        NamesService namesService = ObjectBuilder.buildNamesService();
        List<ProviderFunCall> funCalls = new ArrayList<>();
        ProviderFunCall funCall = new LLMFunCall();
        funCall.setName(namesService.encode(NamesService.PREFIX_TOOLS, "A", "NAME"));
        funCall.setDescription("D");
        funCalls.add(funCall);
        EasyMock.expect(mcpClientService.toolsList(EasyMock.anyString(), EasyMock.anyObject(McpDimension.class))).andReturn(funCalls).anyTimes();
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, McpDimension mcpDimension) throws Exception {
                return mcpDimension;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setNamesService(namesService);
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());
        LLMFunCall llmFunCall = new LLMFunCall();
        Map<String, Object> properties = new HashMap<>();
        llmFunCall.setProperties(Collections.singletonMap("NAME", properties));
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(providerRequest.getMessage()).andReturn(new MessageDelegate(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.replay(mcpClientService, providerRequest);
        providerRequestService.recallMcpFunCalls(providerRequest, new LLMConfig(), llmFunCall);
        Assert.assertEquals(properties, LLMFunCall.class.cast(providerRequestService.recallMcpFunCalls(providerRequest, new LLMConfig(), llmFunCall).getFirst()).getProperties());
        EasyMock.verify(mcpClientService, providerRequest);
    }

    @Test
    public void testFunCallsWithMcpCallException() throws Exception {
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall funCall1 = new LLMFunCall();
        funCall1.setRefer(true);
        funCall1.setName(NamesService.PREFIX_RESOURCE + "OK1");
        funCalls.add(funCall1);
        LLMFunCall funCall2 = new LLMFunCall();
        funCall2.setRefer(true);
        funCall2.setName(NamesService.PREFIX_RESOURCE + "OK2");
        funCalls.add(funCall2);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        OpenAiRequest openAiRequest = new OpenAiRequest();
        providerRequestService.recallFunCall(openAiRequest, new LLMConfig(), funCalls);
        Assert.assertNull(openAiRequest.getFunCalls());
    }

    @Test
    public void testFunCallsWithMcpCallException2() throws Exception {
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall funCall1 = new LLMFunCall();
        funCall1.setRefer(true);
        funCall1.setName(NamesService.PREFIX_RESOURCE + "OK1");
        funCalls.add(funCall1);
        LLMFunCall funCall2 = new LLMFunCall();
        funCall2.setName(NamesService.PREFIX_RESOURCE + "OK2");
        funCalls.add(funCall2);
        OpenAiRequest openAiRequest = new OpenAiRequest();
        openAiRequest.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return openAiRequest;
            }
        };
        providerRequestService.recallFunCall(openAiRequest, new LLMConfig(), funCalls);
        Assert.assertTrue(CollectionUtils.isEmpty(openAiRequest.getFunCalls()));
    }

    @Test
    public void testRecallWorkflowFunCallAndReturnNull() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        EasyMock.expect(workflowConfigService.config("UNKNOWN", "HELLO")).andReturn(workflowConfig);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("HELLO");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        EasyMock.replay(workflowConfigService);
        providerRequestService.recallWorkflowFunCall(request, new LLMConfig(), llmFunCall);
        Assert.assertTrue(CollectionUtils.isEmpty(request.getFunCalls()));
        Assert.assertEquals(workflowConfigService, providerRequestService.getWorkflowConfigService());
        EasyMock.verify(workflowConfigService);
    }

    @Test
    public void testBuildMcpDimension() throws Exception {
        McpDimension mcpDimension = McpDimension.builder().build();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(message);
        McpDimensionService mcpDimensionService = EasyMock.createMock(McpDimensionService.class);
        EasyMock.expect(mcpDimensionService.buildDimension(mcpDimension, message)).andReturn(mcpDimension).anyTimes();
        EasyMock.replay(request, mcpDimensionService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        providerRequestService.setMcpDimensionService(mcpDimensionService);
        Assert.assertEquals(mcpDimension, providerRequestService.buildMcpDimension(request, new LLMConfig(), mcpDimension));
        EasyMock.verify(request, mcpDimensionService);
    }

    @Test
    public void testLoopedFunCall() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setName("BIZ@WR");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        WorkflowConfig workflowConfig2 = new WorkflowConfig();
        LLMFunCall llmFunCall2 = new LLMFunCall();
        llmFunCall2.setName("BIZ@W2R");
        workflowConfig2.setLlmFunCall(llmFunCall2);
        WorkflowConfig workflowConfig3 = new WorkflowConfig();
        LLMFunCall llmFunCall3 = new LLMFunCall();
        llmFunCall3.setName("BIZ@UNKNOWN1");
        workflowConfig3.setLlmFunCall(llmFunCall3);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.expect(workflowConfigService.config("BIZ", "WR2")).andReturn(workflowConfig2).anyTimes();
        EasyMock.expect(workflowConfigService.config("BIZ", "UNKNOWN1")).andReturn(workflowConfig3).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "UNKNOWN1")).andReturn("UNKNOWN_w").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "UNKNOWN2")).andReturn("WRx").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "WR")).andReturn("UNKNOWN_w").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "WR2")).andReturn("UNKNOWN_w").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "W2R")).andReturn("WRx").anyTimes();
        EasyMock.replay(namesService);
        providerRequestService.setNamesService(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("WR");
        funCalls.add(llmFunCall);
        LLMFunCall llmFunCall4 = new LLMFunCall();
        llmFunCall4.setName("UNKNOWN1");
        funCalls.add(llmFunCall4);
        LLMFunCall llmFunCall5 = new LLMFunCall();
        llmFunCall5.setName("BIZ@WR");
        funCalls.add(llmFunCall5);
        LLMFunCall llmFunCall6 = new LLMFunCall();
        llmFunCall6.setName("BIZ@WR2");
        funCalls.add(llmFunCall6);
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        providerRequestService.recallFunCall(request, new LLMConfig(), funCalls);
        Assert.assertEquals("UNKNOWN_w", request.getFunCalls().getFirst().getName());
        Assert.assertEquals("WRx", request.getFunCalls().getLast().getName());
        EasyMock.verify(workflowConfigService, namesService);
    }

    @Test
    public void testNotAllowedWorkflowFunCall() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setName("BIZ@WR");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        WorkflowConfig workflowConfig2 = new WorkflowConfig();
        LLMFunCall llmFunCall2 = new LLMFunCall();
        llmFunCall2.setName("BIZ@W2R");
        workflowConfig2.setLlmFunCall(llmFunCall2);
        WorkflowConfig workflowConfig3 = new WorkflowConfig();
        LLMFunCall llmFunCall3 = new LLMFunCall();
        llmFunCall3.setName("BIZ@UNKNOWN1");
        workflowConfig3.setLlmFunCall(llmFunCall3);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.expect(workflowConfigService.config("BIZ", "WR2")).andReturn(workflowConfig2).anyTimes();
        EasyMock.expect(workflowConfigService.config("BIZ", "UNKNOWN1")).andReturn(workflowConfig3).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "UNKNOWN1")).andReturn("WX").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "UNKNOWN2")).andReturn("W_").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "WR2")).andReturn("WX").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "WR")).andReturn("W_").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "W_")).andReturn("W_").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "BIZ", "W2R")).andReturn("WX_").anyTimes();
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WRX");
        message.setBiz("BIZ");
        request.setMessage(message);
        List<LLMFunCall> funCalls = new ArrayList<LLMFunCall>();
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("WR");
        funCalls.add(llmFunCall);
        LLMFunCall llmFunCall4 = new LLMFunCall();
        llmFunCall4.setName("UNKNOWN1");
        funCalls.add(llmFunCall4);
        LLMFunCall llmFunCall5 = new LLMFunCall();
        llmFunCall5.setName("BIZ@WR");
        funCalls.add(llmFunCall5);
        LLMFunCall llmFunCall6 = new LLMFunCall();
        llmFunCall6.setName("BIZ@WR2");
        funCalls.add(llmFunCall6);
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig_ = new LLMConfig();
        LLMMcpCall mcpConfig = new LLMMcpCall();
        mcpConfig.setBlackList(Arrays.asList("BIZ@UNKNOWN1"));
        llmConfig_.setMcpCall(mcpConfig);
        workflowConfig3.setLlmConfig(llmConfig_);
        providerRequestService.setNamesService(namesService);
        providerRequestService.recallFunCall(request, llmConfig_, funCalls);
        Assert.assertEquals("W_", request.getFunCalls().getFirst().getName());
        Assert.assertEquals("WX_", request.getFunCalls().getLast().getName());
        EasyMock.verify(workflowConfigService, namesService);
    }

    @Test
    public void testTakeoverFunCall() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        EasyMock.expect(workflowConfigService.config("UNKNOWN", "HELLO1")).andReturn(workflowConfig);
        EasyMock.expect(workflowConfigService.config("UNKNOWN", "HELLO2")).andReturn(workflowConfig);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setName("HELLO1");
        LLMTakeover llmTakeover1 = new LLMTakeover();
        llmFunCall1.setTakeover(llmTakeover1);
        LLMFunCall llmFunCall2 = new LLMFunCall();
        llmFunCall2.setName("HELLO2");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        EasyMock.replay(workflowConfigService);
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.expect(namesService.encode("Workflow_", "UNKNOWN", "HELLO1")).andReturn("h1").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "UNKNOWN", "HELLO2")).andReturn("h2").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "UNKNOWN", "h1")).andReturn("h1").anyTimes();
        EasyMock.expect(namesService.encode("Workflow_", "UNKNOWN", "h2")).andReturn("h1").anyTimes();
        EasyMock.replay(namesService);
        providerRequestService.setNamesService(namesService);
        providerRequestService.recallWorkflowFunCall(request, new LLMConfig(), llmFunCall1);
        providerRequestService.recallWorkflowFunCall(request, new LLMConfig(), llmFunCall2);
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(request.getTakeovers().size()));
        Assert.assertEquals(llmTakeover1, request.getTakeover("h1"));
        EasyMock.verify(workflowConfigService, namesService);
    }

    @Test
    public void testWorkflowFunCallDesc1() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setName("BIZ@WR2");
        llmFunCall1.setDescription("DESC");
        llmFunCall1.setPrefix("PREFIX");
        llmFunCall1.setSuffix("SUFFIX");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected String encodeWorkflow(ProviderRequest request, LLMConfig llmConfig, String name, String biz) throws Exception {
                return name;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("BIZ@WR");
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(namesService);
        LLMFunCall recall = providerRequestService.recallWorkflowFunCall(request, llmConfig, llmFunCall);
        Assert.assertEquals("PREFIXDESCSUFFIX", recall.getDescription());
    }

    @Test
    public void testWorkflowFunCallDesc2() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setName("BIZ@WR2");
        llmFunCall1.setDescription("DESC");
        llmFunCall1.setPrefix("PREFIX");
        llmFunCall1.setSuffix("SUFFIX");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected String encodeWorkflow(ProviderRequest request, LLMConfig llmConfig, String name, String biz) throws Exception {
                return name;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("BIZ@WR");
        llmFunCall.setDescription("ABCDE");
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(namesService);
        LLMFunCall recall = providerRequestService.recallWorkflowFunCall(request, llmConfig, llmFunCall);
        Assert.assertEquals("PREFIXDESCSUFFIX", recall.getDescription());
    }

    @Test
    public void testWorkflowFunCallDesc3() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setName("BIZ@WR2");
        llmFunCall1.setPrefix("PREFIX");
        llmFunCall1.setSuffix("SUFFIX");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected String encodeWorkflow(ProviderRequest request, LLMConfig llmConfig, String name, String biz) throws Exception {
                return name;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("BIZ@WR");
        llmFunCall.setDescription("ABCDE");
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(namesService);
        LLMFunCall recall = providerRequestService.recallWorkflowFunCall(request, llmConfig, llmFunCall);
        Assert.assertEquals("PREFIXABCDESUFFIX", recall.getDescription());
    }

    @Test
    public void testWorkflowFunCallDescs1() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setDescriptions(ImmutableMap.of("BIZ@WR", "ABC"));
        llmFunCall1.setName("BIZ@WR2");
        llmFunCall1.setDescription("DESC");
        llmFunCall1.setPrefix("PREFIX");
        llmFunCall1.setSuffix("SUFFIX");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected String encodeWorkflow(ProviderRequest request, LLMConfig llmConfig, String name, String biz) throws Exception {
                return name;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("BIZ@WR");
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(namesService);
        LLMFunCall recall = providerRequestService.recallWorkflowFunCall(request, llmConfig, llmFunCall);
        Assert.assertEquals("PREFIXDESCSUFFIX", recall.getDescription());
    }

    @Test
    public void testWorkflowFunCallDescs2() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setDescriptions(ImmutableMap.of("BIZ@WR", "ABC"));
        llmFunCall1.setName("BIZ@WR2");
        llmFunCall1.setDescription("DESC");
        llmFunCall1.setPrefix("PREFIX");
        llmFunCall1.setSuffix("SUFFIX");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected String encodeWorkflow(ProviderRequest request, LLMConfig llmConfig, String name, String biz) throws Exception {
                return name;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("BIZ@WR");
        llmFunCall.setDescription("ABCD");
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(namesService);
        LLMFunCall recall = providerRequestService.recallWorkflowFunCall(request, llmConfig, llmFunCall);
        Assert.assertEquals("PREFIXDESCSUFFIX", recall.getDescription());
    }

    @Test
    public void testWorkflowFunCallDescs3() throws Exception {
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig1 = new WorkflowConfig();
        LLMFunCall llmFunCall1 = new LLMFunCall();
        llmFunCall1.setDescriptions(ImmutableMap.of("BIZ@WR", "ABC"));
        llmFunCall1.setName("BIZ@WR2");
        llmFunCall1.setPrefix("PREFIX");
        llmFunCall1.setSuffix("SUFFIX");
        workflowConfig1.setLlmFunCall(llmFunCall1);
        EasyMock.expect(workflowConfigService.config("BIZ", "WR")).andReturn(workflowConfig1).anyTimes();
        EasyMock.replay(workflowConfigService);
        ProviderRequestService providerRequestService = new ProviderRequestService() {

            @Override
            protected String encodeWorkflow(ProviderRequest request, LLMConfig llmConfig, String name, String biz) throws Exception {
                return name;
            }

            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        NamesService namesService = EasyMock.createMock(NamesService.class);
        EasyMock.replay(namesService);
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WR");
        message.setBiz("BIZ");
        request.setMessage(message);
        LLMFunCall llmFunCall = new LLMFunCall();
        llmFunCall.setName("BIZ@WR");
        llmFunCall.setDescription("ABCD");
        providerRequestService.setWorkflowConfigService(workflowConfigService);
        LLMConfig llmConfig = new LLMConfig();
        providerRequestService.setNamesService(namesService);
        LLMFunCall recall = providerRequestService.recallWorkflowFunCall(request, llmConfig, llmFunCall);
        Assert.assertEquals("PREFIXABCDSUFFIX", recall.getDescription());
    }

    @Test
    public void testMetadataWithBridgedAndNullMetadata() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setBridged(true);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        llmQuery.getMetadata().clear();
        Map<String, Object> metadata = providerRequestService.buildMetadata(null, llmConfig, llmQuery);
        Assert.assertNotNull(metadata);
        Assert.assertEquals(llmQuery.getUserContext(), metadata.get("userContext"));
    }

    @Test
    public void testRecallWorkflowFunCallFilteredByMcp() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        WorkflowConfigService workflowConfigService = EasyMock.createMock(WorkflowConfigService.class);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        EasyMock.expect(workflowConfigService.config(EasyMock.anyString(), EasyMock.anyString())).andReturn(workflowConfig).anyTimes();
        providerRequestService.setWorkflowConfigService(workflowConfigService);

        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setBlackList(Arrays.asList("UNKNOWN@HELLO"));
        llmConfig.setMcpCall(mcpCall);

        LLMFunCall funCall = new LLMFunCall();
        funCall.setName("HELLO");

        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));

        EasyMock.replay(workflowConfigService);
        LLMFunCall result = providerRequestService.recallWorkflowFunCall(request, llmConfig, funCall);
        Assert.assertNull(result);
        EasyMock.verify(workflowConfigService);
    }

    @Test
    public void testRecallMcpFunCallsEmpty() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig) {
                return null;
            }
        };
        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        EasyMock.expect(mcpClientService.toolsList(EasyMock.anyString(), EasyMock.anyObject())).andReturn(Collections.emptyList());
        providerRequestService.setMcpClientService(mcpClientService);

        LLMFunCall funCall = new LLMFunCall();
        funCall.setName("MCP_TOOL");
        GoogleRequest request = new GoogleRequest();

        EasyMock.replay(mcpClientService);
        Assert.assertNull(providerRequestService.recallMcpFunCalls(request, new LLMConfig(), funCall));
    }

    @Test
    public void testReConfigMcpFunCallWithProperties() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        ProviderFunCall mcpCall = new LLMFunCall();
        LLMFunCall funCall = new LLMFunCall();
        Map<String, Object> props = Collections.singletonMap("key", "value");
        funCall.setProperties(Collections.singletonMap("tool", props));

        providerRequestService.reConfigMcpFunCall(mcpCall, new LLMConfig(), funCall, "tool");
        Assert.assertEquals(props, mcpCall.getProperties());
    }

    @Test
    public void testExternalHistoryNotReplace() throws Exception {
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return null;
            }

            @Override
            protected McpDimension buildMcpDimension(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return McpDimension.builder().mcpConfig(new McpConfig()).build();
            }

            @Override
            protected McpRuntime buildMcpRuntime(ProviderRequest request, LLMConfig llmConfig, LLMMcpCall llmMcpCall) {
                return null;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        LLMMcpCall mcpCall = new LLMMcpCall();
        mcpCall.setClient("CLIENT");
        mcpCall.setReplace(true);
        llmConfig.setMcpCall(mcpCall);

        McpClientServiceImpl mcpClientService = EasyMock.createMock(McpClientServiceImpl.class);
        List<History> histories = Arrays.asList(new History(), new History());
        McpResult<List<History>> result = new McpResult<>();
        result.setResult(histories);
        EasyMock.expect(mcpClientService.promptGet(EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyObject(), EasyMock.anyObject())).andReturn(result);

        providerRequestService.setMcpClientService(mcpClientService);
        providerRequestService.setMcpTriggerService(new McpTriggerServiceImpl());
        providerRequestService.setMcpRewriteService(new McpRewriteServiceImpl());

        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);

        EasyMock.replay(mcpClientService);
        providerRequestService.externalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals("UNKNOWN", message.getQuery()); // Should not be replaced
        EasyMock.verify(mcpClientService);
    }

    @Test
    public void testBuildTimeoutWhenUpstreamNotEmptyAndDeepnessOne() throws Exception {
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        service.setFunCallTimeout(5000);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(10000);
        llmConfig.setFunCallWaiting(30000);
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.getUpstream()).andReturn("some-upstream").anyTimes();
        EasyMock.expect(llmQuery.getDeepness()).andReturn(1).anyTimes();
        EasyMock.expect(llmQuery.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(llmQuery.isEntry()).andReturn(false).anyTimes();
        EasyMock.replay(llmQuery);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        Integer result = service.buildFunCallTimeout(request, llmConfig, llmQuery);
        Assert.assertEquals(Integer.valueOf(5000), result);
        EasyMock.verify(request, llmQuery);
    }

    @Test
    public void testBuildTimeoutWhenUpstreamEmpty() throws Exception {
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        service.setFunCallTimeout(5000);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(10000);
        llmConfig.setFunCallWaiting(30000);
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.getUpstream()).andReturn("").anyTimes();
        EasyMock.expect(llmQuery.getDeepness()).andReturn(1).anyTimes();
        EasyMock.expect(llmQuery.getMetadata()).andReturn(Collections.emptyMap()).anyTimes();
        EasyMock.expect(llmQuery.isEntry()).andReturn(false).anyTimes();
        EasyMock.replay(llmQuery);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        Integer result = service.buildFunCallTimeout(request, llmConfig, llmQuery);
        Assert.assertEquals(Integer.valueOf(5000), result);
        EasyMock.verify(request, llmQuery);
    }

    @Test
    public void testBuildTimeoutWhenDeepnessNotOne() throws Exception {
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        service.setFunCallTimeout(5000);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(10000);
        llmConfig.setFunCallWaiting(30000);
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(llmQuery.getUpstream()).andReturn("some-upstream").anyTimes();
        EasyMock.expect(llmQuery.getDeepness()).andReturn(2).anyTimes();
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.isEntry()).andReturn(false).anyTimes();
        EasyMock.replay(llmQuery);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        Integer result = service.buildFunCallTimeout(request, llmConfig, llmQuery);
        Assert.assertEquals(Integer.valueOf(5000), result);
        EasyMock.verify(request, llmQuery);
    }

    @Test
    public void testBuildTimeoutUsesDefaultTimeoutWhenConfigTimeoutNull() throws Exception {
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        service.setFunCallTimeout(8000);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setFunCallWaiting(20000);
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(llmQuery.getUpstream()).andReturn("up").anyTimes();
        EasyMock.expect(llmQuery.getDeepness()).andReturn(1).anyTimes();
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.isEntry()).andReturn(true).anyTimes();
        EasyMock.replay(llmQuery);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        Integer result = service.buildFunCallTimeout(request, llmConfig, llmQuery);
        Assert.assertEquals(Integer.valueOf(20000), result);
        EasyMock.verify(request, llmQuery);
    }

    @Test
    public void testBuildTimeoutWhenUpstreamNull() throws Exception {
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return null;
            }
        };
        service.setFunCallTimeout(5000);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(10000);
        LLMQuery llmQuery = EasyMock.createMock(LLMQuery.class);
        EasyMock.expect(llmQuery.getUpstream()).andReturn(null).anyTimes();
        EasyMock.expect(llmQuery.getDeepness()).andReturn(1).anyTimes();
        EasyMock.expect(llmQuery.getMetadata()).andReturn(new HashMap<>()).anyTimes();
        EasyMock.expect(llmQuery.containHistories()).andReturn(false).anyTimes();
        EasyMock.expect(llmQuery.isEntry()).andReturn(true).anyTimes();
        EasyMock.replay(llmQuery);
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        EasyMock.expect(request.getMessage()).andReturn(new MessageDelegate(llmQuery)).anyTimes();
        EasyMock.replay(request);
        Integer result = service.buildFunCallTimeout(request, llmConfig, llmQuery);
        Assert.assertEquals(Integer.valueOf(10000), result);
        EasyMock.verify(request, llmQuery);
    }

    @Test
    public void testStoreRequest() throws Exception {
        ProviderRequest request = EasyMock.createMock(ProviderRequest.class);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        EasyMock.expect(request.getStoreQuery()).andReturn(true).anyTimes();
        EasyMock.expect(request.getQuery4History()).andReturn("HELLO").anyTimes();
        EasyMock.expect(request.getStoreCompleted()).andReturn(false).anyTimes();
        EasyMock.expect(request.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(request.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(request.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(request.getMessage()).andReturn(message).anyTimes();
        EasyMock.expect(request.getScene()).andReturn("WORKFLOW").anyTimes();
        EasyMock.expect(request.getClientHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(request.getHistories()).andReturn(10).anyTimes();
        EasyMock.expect(request.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(request.getApi()).andReturn("WORLD").anyTimes();
        message.setWorkflow("WORKFLOW");
        EasyMock.replay(request);
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        List<HistoryPair> mockHistories = new ArrayList<>();
        h.store(request.getMessage(), request.getRepositories(), mockHistories, request.getExpired(), request.getHistories());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(h);
        ProviderRequestService providerRequestService = new ProviderRequestService() {
            @Override
            protected ProviderRequest build() {
                return request;
            }

            @Override
            protected List<HistoryPair> buildHistoryQuery(ProviderRequest request, LLMConfig llmConfig) throws Exception {
                List<HistoryPair> historyPairs = super.buildHistoryQuery(request, llmConfig);
                Assert.assertEquals("HELLO", historyPairs.getFirst().getQuery());
                Assert.assertEquals(Long.valueOf(request.getMessage().getCreated() + 1), Long.valueOf(historyPairs.getFirst().getCreated()));
                return mockHistories;
            }
        };
        providerRequestService.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        providerRequestService.setHistoryStore(h);
        providerRequestService.storeHistoryQuery(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertNotNull(providerRequestService.getProviderRequestRewriter());
        EasyMock.verify(request, h);
    }

    @Test
    public void testBuildHistoryQuery() throws Exception {
        long timestamp = 100086L;
        String conversation = "conv-001";
        String chat = "chat-002";
        String query4History = "query-for-history";
        NettyRequest wt = (NettyRequest) ObjectBuilder.buildWorkflowTaskWithTimestamp(timestamp);
        wt.setConversation(conversation);
        wt.setChat(chat);
        wt.setWorkflow("WORKFLOW");
        wt.setBiz("BIZ");
        wt.setQuery(query4History);
        Message message = Message.build(LLMQuery.build(wt));
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        request.setPureQuery(false);
        request.setModel("m-req");
        request.setApi("api-req");
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        LLMConfig llmConfig = new LLMConfig();
        List<HistoryPair> result = service.buildHistoryQuery(request, llmConfig);
        Assert.assertNotNull(result);
        Assert.assertEquals(1, result.size());
        HistoryPair pair = result.get(0);
        Assert.assertEquals(Long.valueOf(timestamp + 1), pair.getCreated());
        Assert.assertEquals(conversation, pair.getConversation());
        Assert.assertEquals(chat, pair.getChat());
        Assert.assertEquals(query4History, pair.getQuery());
        Assert.assertEquals("m-req", pair.getModel());
        Assert.assertEquals("api-req", pair.getApi());
    }

    /**
     * 覆盖 buildContainHistories：metadata 含 KEY_CONTAIN_HISTORIES=true 时返回 true
     */
    @Test
    public void testBuildContainHistories_fromMetadataTrue() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__containHistories", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setContainHistories(false);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        Boolean result = service.buildContainHistories(request, llmConfig, llmQuery);
        Assert.assertTrue(result);
    }

    /**
     * 覆盖 buildContainHistories：metadata 含 KEY_CONTAIN_HISTORIES=false 时返回 false
     */
    @Test
    public void testBuildContainHistories_fromMetadataFalse() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__containHistories", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setContainHistories(true);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        Boolean result = service.buildContainHistories(request, llmConfig, llmQuery);
        Assert.assertFalse(result);
    }

    /**
     * 覆盖 buildContainHistories：metadata 无 KEY_CONTAIN_HISTORIES 时取 llmConfig.getContainHistories()
     */
    @Test
    public void testBuildContainHistories_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setContainHistories(true);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        Boolean result = service.buildContainHistories(request, llmConfig, llmQuery);
        Assert.assertTrue(result);
        llmConfig.setContainHistories(false);
        result = service.buildContainHistories(request, llmConfig, llmQuery);
        Assert.assertFalse(result);
    }

    private static ProviderRequestService<ProviderRequest> createService(ProviderRequest request) {
        return new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
    }

    @Test
    public void testBuildModel_readsMetadataFromLlmQuery() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery(ImmutableMap.of("__model", "message-model"))));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__model", "query-model"));
        LLMConfig llmConfig = new LLMConfig();

        Assert.assertEquals("query-model", createService(request).buildModel(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildToken_readsMetadataFromLlmQuery() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery(ImmutableMap.of("__token", "message-token"))));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__token", "query-token"));
        LLMConfig llmConfig = new LLMConfig();
        ProviderRequestService<ProviderRequest> service = createService(request);
        service.setProviderToken(new ProviderToken());

        Assert.assertEquals("query-token", service.buildToken(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildScene_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__scene", "SCENE_FROM_METADATA"));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setScene("config-scene");
        Assert.assertEquals("SCENE_FROM_METADATA", createService(request).buildScene(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildScene_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setScene("config-scene");
        Assert.assertEquals("config-scene", createService(request).buildScene(request, llmConfig, llmQuery));
    }

    /**
     * buildUrl：metadata 含 __url 时原样返回
     */
    @Test
    public void testBuildUrl_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__url", "https://api.example.com/v1/chat"));
        LLMConfig llmConfig = new LLMConfig();
        Assert.assertEquals("https://api.example.com/v1/chat", createService(request).buildUrl(request, llmConfig, llmQuery));
    }

    /**
     * buildUrl：无 __url 时 MapUtils.getString 返回 null
     */
    @Test
    public void testBuildUrl_missingKeyReturnsNull() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        Assert.assertNull(createService(request).buildUrl(request, llmConfig, llmQuery));
    }

    /**
     * buildUrl：__url 为空串时返回空串
     */
    @Test
    public void testBuildUrl_emptyString() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__url", ""));
        LLMConfig llmConfig = new LLMConfig();
        Assert.assertEquals("", createService(request).buildUrl(request, llmConfig, llmQuery));
    }

    /**
     * buildPrompt：metadata 含 __prompt 时写入该字符串（注意：Java 会先求值默认参数，llmPromptService.prompt 仍会被调用一次）。
     */
    @Test
    public void testBuildPrompt_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__prompt", "metadata-override-prompt"));
        LLMConfig llmConfig = new LLMConfig();
        LLMPromptService promptService = EasyMock.createMock(LLMPromptService.class);
        EasyMock.expect(promptService.prompt(new OpenAiRequest(), llmConfig, llmQuery)).andReturn("from-llm-prompt-service").once();
        EasyMock.replay(promptService);
        ProviderRequestService<ProviderRequest> service = createService(request);
        service.setLlmPromptService(promptService);
        service.buildPrompt(request, llmConfig, llmQuery);
        Assert.assertEquals("metadata-override-prompt", request.getPrompt());
    }

    /**
     * buildPrompt：metadata 无 __prompt 时使用 llmPromptService.prompt 结果。
     */
    @Test
    public void testBuildPrompt_defaultFromLlmPromptService() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        LLMPromptService promptService = EasyMock.createMock(LLMPromptService.class);
        EasyMock.expect(promptService.prompt(request, llmConfig, llmQuery)).andReturn("dynamic-prompt").once();
        EasyMock.replay(promptService);
        ProviderRequestService<ProviderRequest> service = createService(request);
        service.setLlmPromptService(promptService);
        service.buildPrompt(request, llmConfig, llmQuery);
        Assert.assertEquals("dynamic-prompt", request.getPrompt());
        EasyMock.verify(promptService);
    }

    @Test
    public void testBuildTimeout_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__timeout", 42000));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(10000);
        ProviderRequestService<ProviderRequest> service = createService(request);
        Assert.assertEquals(Integer.valueOf(42000), service.buildTimeout(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildTimeout_defaultFromLlmConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(15000);
        ProviderRequestService<ProviderRequest> service = createService(request);
        Assert.assertEquals(Integer.valueOf(15000), service.buildTimeout(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildTimeout_fallbackToServiceInjectedTimeout() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setTimeout(88888);
        ProviderRequestService<ProviderRequest> service = createService(request);
        Assert.assertEquals(Integer.valueOf(88888), service.buildTimeout(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildClientHistories_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__clientHistories", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientHistories(false);
        Assert.assertTrue(createService(request).buildClientHistories(request, llmConfig, llmQuery));
    }

    /**
     * 覆盖 buildClientDowngrade：metadata __clientDowngrade=true 时覆盖 llmConfig
     */
    @Test
    public void testBuildClientDowngrade_fromMetadataTrue() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__clientDowngrade", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientDowngrade(false);
        Assert.assertTrue(createService(request).buildClientDowngrade(request, llmConfig, llmQuery));
    }

    /**
     * 覆盖 buildClientDowngrade：metadata __clientDowngrade=false 时覆盖 llmConfig
     */
    @Test
    public void testBuildClientDowngrade_fromMetadataFalse() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__clientDowngrade", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientDowngrade(true);
        Assert.assertFalse(createService(request).buildClientDowngrade(request, llmConfig, llmQuery));
    }

    /**
     * 覆盖 buildClientDowngrade：metadata 无 __clientDowngrade 时取 llmConfig.getClientDowngrade()
     */
    @Test
    public void testBuildClientDowngrade_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientDowngrade(true);
        Assert.assertTrue(createService(request).buildClientDowngrade(request, llmConfig, llmQuery));
        llmConfig.setClientDowngrade(false);
        Assert.assertFalse(createService(request).buildClientDowngrade(request, llmConfig, llmQuery));
    }

    /**
     * request() 中会调用 setClientDowngrade(buildClientDowngrade(...))，metadata 覆盖配置
     */
    @Test
    public void testRequest_appliesClientDowngradeFromMetadata() throws Exception {
        Map<String, Object> meta = new HashMap<>();
        meta.put("__clientDowngrade", false);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(meta);
        Message message = Message.build(llmQuery);
        message.setNotifier("N");
        GoogleRequest request = new GoogleRequest();
        request.setMessage(message);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientDowngrade(true);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected void buildPrompt(ProviderRequest r, LLMConfig c, LLMQuery q) throws Exception {
            }

            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setProviderRequestRewriter(new ProviderRequestRewriter.BaseRequestRewriter());
        service.setProviderToken(new ProviderToken());
        service.request(request, llmConfig, message);
        Assert.assertFalse(request.getClientDowngrade());
    }

    @Test
    public void testBuildStoreCompleted_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__storeCompleted", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setStoreCompleted(true);
        Assert.assertFalse(createService(request).buildStoreCompleted(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildRepositories_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        List<String> metaRepos = Arrays.asList("repo-a", "repo-b");
        Map<String, Object> meta = new HashMap<>();
        meta.put("__repositories", metaRepos);
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(meta);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRepositories(Arrays.asList("config-repo"));
        List<String> result = createService(request).buildRepositories(request, llmConfig, llmQuery);
        Assert.assertEquals(metaRepos, result);
    }

    @Test
    public void testBuildRepositories_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        List<String> configRepos = Arrays.asList("only-repo");
        llmConfig.setRepositories(configRepos);
        Assert.assertEquals(configRepos, createService(request).buildRepositories(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildStoreFunCall_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__storeFunCall", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setStoreFunCall(false);
        Assert.assertTrue(createService(request).buildStoreFunCall(request, llmConfig, llmQuery));
    }

    /**
     * 覆盖 buildRecallFunCall：metadata 含 __recallFunCall=true 时返回 true
     */
    @Test
    public void testBuildRecallFunCall_fromMetadataTrue() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__recallFunCall", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRecallFunCall(false);
        Assert.assertTrue(createService(request).buildRecallFunCall(request, llmConfig, llmQuery));
    }

    /**
     * 覆盖 buildRecallFunCall：metadata 含 __recallFunCall=false 时返回 false
     */
    @Test
    public void testBuildRecallFunCall_fromMetadataFalse() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__recallFunCall", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRecallFunCall(true);
        Assert.assertFalse(createService(request).buildRecallFunCall(request, llmConfig, llmQuery));
    }

    /**
     * 覆盖 buildRecallFunCall：metadata 无 __recallFunCall 时取 llmConfig.getRecallFunCall()
     */
    @Test
    public void testBuildRecallFunCall_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setRecallFunCall(true);
        Assert.assertTrue(createService(request).buildRecallFunCall(request, llmConfig, llmQuery));
        llmConfig.setRecallFunCall(false);
        Assert.assertFalse(createService(request).buildRecallFunCall(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildPrintReason_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__printReason", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setPrintReason(true);
        Assert.assertFalse(createService(request).buildPrintReason(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildWriteable_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__writeable", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setWriteable(false);
        Assert.assertTrue(createService(request).buildWriteable(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildHistories_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__histories", 20));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        Assert.assertEquals(Integer.valueOf(20), createService(request).buildHistories(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildHistories_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(15);
        Assert.assertEquals(Integer.valueOf(15), createService(request).buildHistories(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildMaxError_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__maxError", 2048));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setMaxError(1024);
        Assert.assertEquals(Integer.valueOf(2048), createService(request).buildMaxError(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildNotifier_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__notifier", "custom-notifier"));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setNotifier("config-notifier");
        Assert.assertEquals("custom-notifier", createService(request).buildNotifier(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildExpired_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__expired", 3600));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setExpired(1800);
        Assert.assertEquals(Integer.valueOf(3600), createService(request).buildExpired(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildExpired_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setExpired(7200);
        Assert.assertEquals(Integer.valueOf(7200), createService(request).buildExpired(request, llmConfig, llmQuery));
    }

    /**
     * internalHistory：云端恢复为空 + ClientDowngrade + 端侧 REFERENCE_CLIENT 非空时，用端侧记忆替换
     */
    @Test
    public void internalHistory_restoreEmpty_clientDowngradeTrue_usesClientReferenceHistories() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(true);
        request.setRecallFunCall(true);
        History clientH = new History();
        clientH.setReference(History.REFERENCE_CLIENT);
        clientH.setContent("from-client");
        message.addHistory(clientH);
        setHistoryStrictlyBeforeMessage(message, clientH);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        Assert.assertEquals(1, out.size());
        Assert.assertEquals("from-client", out.get(0).getContent());
        Assert.assertEquals(History.REFERENCE_CLIENT, out.get(0).getReference());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：云端为空且 ClientDowngrade=false 时，不做端侧降级替换（结果为空）
     */
    @Test
    public void internalHistory_restoreEmpty_clientDowngradeFalse_staysEmpty() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(false);
        History clientH = new History();
        clientH.setReference(History.REFERENCE_CLIENT);
        clientH.setContent("from-client");
        message.addHistory(clientH);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertTrue(request.getMessage().getHistories().isEmpty());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：云端有记忆时优先使用云端，不因 ClientDowngrade 改用端侧备份
     */
    @Test
    public void internalHistory_restoreNonEmpty_prefersCloudOverClientDowngradeBackup() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(true);
        History clientH = new History();
        clientH.setReference(History.REFERENCE_CLIENT);
        clientH.setContent("client-backup");
        message.addHistory(clientH);
        History cloud = new History();
        cloud.setReference(History.REFERENCE_SERVER);
        cloud.setContent("from-cloud");
        setHistoryStrictlyBeforeMessage(message, cloud);
        List<History> cloudList = new ArrayList<>(Collections.singletonList(cloud));
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(cloudList).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        Assert.assertEquals(1, out.size());
        Assert.assertEquals("from-cloud", out.get(0).getContent());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：云端为空、ClientDowngrade 开启但消息中无 REFERENCE_CLIENT 时，无法降级替换
     */
    @Test
    public void internalHistory_restoreEmpty_clientDowngradeTrue_noClientReference_staysEmpty() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(true);
        History serverOnly = new History();
        serverOnly.setReference(History.REFERENCE_SERVER);
        serverOnly.setContent("server-only");
        message.addHistory(serverOnly);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertTrue(request.getMessage().getHistories().isEmpty());
        EasyMock.verify(historyStore);
    }

    @Test
    public void isFromFunMerge_nullMetadata_returnsFalse() {
        Assert.assertFalse(ProviderRequestService.isFromFunMerge(null));
    }

    @Test
    public void isFromFunMerge_emptyOrWithoutKey_returnsFalse() {
        Assert.assertFalse(ProviderRequestService.isFromFunMerge(new HashMap<>()));
        Map<String, Object> m = new HashMap<>();
        m.put(ProviderRequestService.KEY_FUN_FETCH, new Object());
        Assert.assertFalse(ProviderRequestService.isFromFunMerge(m));
    }

    @Test
    public void isFromFunMerge_withMergeKey_returnsTrue() {
        Map<String, Object> m = new HashMap<>();
        m.put(ProviderRequestService.KEY_FUN_MERGE, new Object());
        Assert.assertTrue(ProviderRequestService.isFromFunMerge(m));
    }

    @Test
    public void testBuildDiscard_fromMetadata() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__discard", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setDiscard(true);
        Assert.assertFalse(createService(request).buildDiscard(request, llmConfig, llmQuery));
    }

    @Test
    public void testBuildDiscard_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setDiscard(false);
        Assert.assertFalse(createService(request).buildDiscard(request, llmConfig, llmQuery));
    }

    @Test
    public void discard_recallFunCallFalse_leavesHistoriesUnchanged() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setRecallFunCall(false);
        request.setApi(ProviderRequest.REQUEST_OPENAI);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        History incompatible = funCallHistory("anthropic");
        message.replaceHistories(new ArrayList<>(Collections.singletonList(incompatible)));
        new DiscardHookService().runDiscard(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(1, message.getHistories().size());
        Assert.assertEquals("anthropic", message.getHistories().get(0).getApi());
    }

    @Test
    public void discard_recallFunCallTrue_emptyHistories_noOp() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setRecallFunCall(true);
        request.setDiscard(true);
        request.setApi(ProviderRequest.REQUEST_OPENAI);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        message.replaceHistories(new ArrayList<>());
        new DiscardHookService().runDiscard(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertTrue(message.getHistories().isEmpty());
    }

    @Test
    public void discard_recallFunCallTrue_removesOnlyIncompatibleFunCallApi() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setRecallFunCall(true);
        request.setDiscard(true);
        request.setApi(ProviderRequest.REQUEST_OPENAI);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        History chat = new History();
        chat.setFunction(History.FUN_CHAT);
        chat.setApi("anthropic");
        chat.setContent("chat");
        History defApi = funCallHistory(ProviderRequest.REQUEST_DEF);
        defApi.setContent("def-funcall");
        History sameApi = funCallHistory(ProviderRequest.REQUEST_OPENAI);
        sameApi.setContent("openai-funcall");
        History otherApi = funCallHistory("anthropic");
        otherApi.setContent("drop-me");
        message.replaceHistories(new ArrayList<>(Arrays.asList(chat, defApi, sameApi, otherApi)));
        new DiscardHookService().runDiscard(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        List<History> left = message.getHistories();
        Assert.assertEquals(3, left.size());
        Assert.assertTrue(left.stream().anyMatch(h -> "chat".equals(h.getContent())));
        Assert.assertTrue(left.stream().anyMatch(h -> "def-funcall".equals(h.getContent())));
        Assert.assertTrue(left.stream().anyMatch(h -> "openai-funcall".equals(h.getContent())));
        Assert.assertFalse(left.stream().anyMatch(h -> "drop-me".equals(h.getContent())));
    }

    @Test
    public void discard_recallFunCallTrue_discardFalse_doesNotRemoveHistories() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setRecallFunCall(true);
        request.setDiscard(false);
        request.setApi(ProviderRequest.REQUEST_OPENAI);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        History otherApi = funCallHistory("anthropic");
        otherApi.setContent("keep-incompatible");
        message.replaceHistories(new ArrayList<>(Collections.singletonList(otherApi)));
        new DiscardHookService().runDiscard(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(1, message.getHistories().size());
        Assert.assertEquals("keep-incompatible", message.getHistories().get(0).getContent());
    }

    @Test
    public void discard_apiMatchIsCaseInsensitive() throws Exception {
        OpenAiRequest request = new OpenAiRequest();
        request.setRecallFunCall(true);
        request.setDiscard(true);
        request.setApi(ProviderRequest.REQUEST_OPENAI);
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        request.setMessage(message);
        History mixedCase = funCallHistory("OpenAI");
        mixedCase.setContent("keep");
        message.replaceHistories(new ArrayList<>(Collections.singletonList(mixedCase)));
        new DiscardHookService().runDiscard(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(1, message.getHistories().size());
        Assert.assertEquals("keep", message.getHistories().get(0).getContent());
    }

    private static History funCallHistory(String api) {
        History h = new History();
        h.setFunction(History.FUN_FUNCALL);
        h.setApi(api);
        return h;
    }

    /**
     * internalHistory：containHistories=false 时，不恢复记忆，ClientDowngrade 不生效
     */
    @Test
    public void internalHistory_containHistoriesFalse_clientDowngradeTrue_noRestore() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(false);
        request.setClientHistories(false);
        request.setClientDowngrade(true);
        request.setRecallFunCall(true);
        History clientH = new History();
        clientH.setReference(History.REFERENCE_CLIENT);
        clientH.setContent("from-client");
        message.addHistory(clientH);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        // historyStore 不应被调用
        service.setHistoryStore(EasyMock.createMock(HistoryStore.class));
        service.internalHistory(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        // containHistories=false，不进入恢复逻辑，histories 被 delHistories 清空
        Assert.assertTrue(CollectionUtils.isEmpty(request.getMessage().getHistories()));
    }

    /**
     * internalHistory：clientHistories=true 时跳过开头 delHistories；云端为空且 ClientDowngrade 时用 REFERENCE_CLIENT 备份经 replaceHistories 写回（非与云端合并）
     */
    @Test
    public void internalHistory_clientHistoriesTrue_clientDowngradeTrue_restoreEmpty_usesClientHistories() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(true);
        request.setClientDowngrade(true);
        request.setRecallFunCall(true);
        History clientH = new History();
        clientH.setReference(History.REFERENCE_CLIENT);
        clientH.setContent("client-kept");
        message.addHistory(clientH);
        setHistoryStrictlyBeforeMessage(message, clientH);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        Assert.assertEquals(1, out.size());
        Assert.assertEquals("client-kept", out.get(0).getContent());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：clientHistories=true 且云端非空时，replaceHistories 整表替换，原 message 上端侧记忆被覆盖
     */
    @Test
    public void internalHistory_clientHistoriesTrue_cloudNonEmpty_replacesEntireMessageHistories() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(true);
        request.setClientDowngrade(true);
        request.setRecallFunCall(true);
        History preClient = new History();
        preClient.setReference(History.REFERENCE_CLIENT);
        preClient.setContent("pre-client-only");
        message.addHistory(preClient);
        History cloud = new History();
        cloud.setReference(History.REFERENCE_SERVER);
        cloud.setContent("from-cloud");
        setHistoryStrictlyBeforeMessage(message, preClient, cloud);
        List<History> cloudList = new ArrayList<>(Collections.singletonList(cloud));
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(cloudList).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        Assert.assertEquals(1, out.size());
        Assert.assertEquals("from-cloud", out.get(0).getContent());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：clientHistories=true、云端为空且未开启 ClientDowngrade 时，replaceHistories(空) 清空原端侧记忆
     */
    @Test
    public void internalHistory_clientHistoriesTrue_restoreEmpty_clientDowngradeFalse_clearsPreClient() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(true);
        request.setClientDowngrade(false);
        request.setRecallFunCall(true);
        History preClient = new History();
        preClient.setReference(History.REFERENCE_CLIENT);
        preClient.setContent("lost-without-downgrade");
        message.addHistory(preClient);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertTrue(request.getMessage().getHistories().isEmpty());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：六参 restore 与 Redis score 一致——now=-timestamp，end=-(timestamp-recallOffset)。
     */
    @Test
    public void internalHistory_restore_sixArgs_nowAndEndMatchScoreConvention() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setRecallOffset(0);
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(false);
        request.setRecallFunCall(true);
        Capture<Long> nowArg = EasyMock.newCapture();
        Capture<Long> endArg = EasyMock.newCapture();
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(
                        EasyMock.eq(message),
                        EasyMock.eq("SCENE"),
                        EasyMock.anyInt(),
                        EasyMock.anyBoolean(),
                        EasyMock.capture(nowArg),
                        EasyMock.capture(endArg)))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        long ts = message.getCreated();
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        Assert.assertEquals(Long.valueOf(-ts), nowArg.getValue());
        Assert.assertEquals(Long.valueOf(-(ts)), endArg.getValue());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：剔除 created &gt;= message.timestamp 的恢复项（避免与当前轮/FunCall 重叠）
     */
    @Test
    public void internalHistory_removesHistoriesNotStrictlyBeforeMessageTimestamp() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        long ts = message.getCreated();
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(false);
        request.setRecallFunCall(true);
        History oldEnough = new History();
        oldEnough.setContent("keep");
        oldEnough.setCreated(ts - 10L);
        History sameTs = new History();
        sameTs.setContent("drop-same-ts");
        sameTs.setCreated(ts);
        History future = new History();
        future.setContent("drop-future");
        future.setCreated(ts + 100L);
        List<History> restored = new ArrayList<>(Arrays.asList(oldEnough, sameTs, future));
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(restored).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        Assert.assertEquals(1, out.size());
        Assert.assertEquals("keep", out.get(0).getContent());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：ClientDowngrade=true + 云端为空 + 端侧有多条 CLIENT 记忆时，全部替换
     */
    @Test
    public void internalHistory_restoreEmpty_clientDowngradeTrue_multipleClientHistories() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(true);
        request.setRecallFunCall(true);
        History c1 = new History();
        c1.setReference(History.REFERENCE_CLIENT);
        c1.setContent("client-1");
        History c2 = new History();
        c2.setReference(History.REFERENCE_CLIENT);
        c2.setContent("client-2");
        History s1 = new History();
        s1.setReference(History.REFERENCE_SERVER);
        s1.setContent("server-1");
        message.addHistory(c1);
        message.addHistory(c2);
        message.addHistory(s1);
        setHistoryStrictlyBeforeMessage(message, c1, c2, s1);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        // 仅 CLIENT 记忆被备份并替换，SERVER 记忆不在备份中
        Assert.assertEquals(2, out.size());
        Assert.assertEquals("client-1", out.get(0).getContent());
        Assert.assertEquals("client-2", out.get(1).getContent());
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：ClientDowngrade=true + 云端为空 + recallFunCall=false + clientHistories=false 时，
     * 降级使用 CLIENT 备份，但其中 FUN_FUNCALL 仍会被剔除
     */
    @Test
    public void internalHistory_restoreEmpty_clientDowngradeTrue_recallFunCallFalse_clientHistoriesFalse() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(true);
        request.setClientHistories(false);
        request.setClientDowngrade(true);
        request.setRecallFunCall(false);
        History clientChat = new History();
        clientChat.setReference(History.REFERENCE_CLIENT);
        clientChat.setContent("client-chat");
        clientChat.setFunction(History.FUN_CHAT);
        History clientFunCall = new History();
        clientFunCall.setReference(History.REFERENCE_CLIENT);
        clientFunCall.setContent("client-funcall");
        clientFunCall.setFunction(History.FUN_FUNCALL);
        message.addHistory(clientChat);
        message.addHistory(clientFunCall);
        setHistoryStrictlyBeforeMessage(message, clientChat, clientFunCall);
        HistoryStore historyStore = EasyMock.createMock(HistoryStore.class);
        EasyMock.expect(historyStore.restore(EasyMock.eq(message), EasyMock.eq("SCENE"), EasyMock.anyInt(), EasyMock.anyBoolean(), EasyMock.anyLong(), EasyMock.anyLong()))
                .andReturn(Collections.emptyList()).once();
        EasyMock.replay(historyStore);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(historyStore);
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setHistories(10);
        llmConfig.setRecallNums(10);
        llmConfig.setRecallDesc(false);
        service.internalHistory(request, llmConfig, ObjectBuilder.buildLLMQuery());
        List<History> out = request.getMessage().getHistories();
        Assert.assertEquals(1, out.size());
        Assert.assertEquals("client-chat", out.get(0).getContent());
        Assert.assertFalse(out.stream().anyMatch(h -> "client-funcall".equals(h.getContent())));
        EasyMock.verify(historyStore);
    }

    /**
     * internalHistory：containHistories=false + clientDowngrade=false 时，清空端侧记忆且不恢复
     */
    @Test
    public void internalHistory_containHistoriesFalse_clientDowngradeFalse_clearsAll() throws Exception {
        GoogleRequest request = new GoogleRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setWorkflow("WF");
        request.setMessage(message);
        request.setScene("SCENE");
        request.setContainHistories(false);
        request.setClientHistories(false);
        request.setClientDowngrade(false);
        request.setRecallFunCall(true);
        History h = new History();
        h.setContent("will-be-cleared");
        message.addHistory(h);
        ProviderRequestService<ProviderRequest> service = new ProviderRequestService<ProviderRequest>() {
            @Override
            protected ProviderRequest build() {
                return request;
            }
        };
        service.setHistoryStore(EasyMock.createMock(HistoryStore.class));
        service.internalHistory(request, new LLMConfig(), ObjectBuilder.buildLLMQuery());
        Assert.assertTrue(CollectionUtils.isEmpty(request.getMessage().getHistories()));
    }

    /**
     * buildClientDowngrade：metadata 为 null 时取 llmConfig 默认值
     */
    @Test
    public void testBuildClientDowngrade_nullMetadata_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientDowngrade(true);
        Assert.assertTrue(createService(request).buildClientDowngrade(request, llmConfig, llmQuery));
    }

    /**
     * buildClientHistories：metadata __clientHistories=true 时覆盖 llmConfig
     */
    @Test
    public void testBuildClientHistories_fromMetadataTrue() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__clientHistories", true));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientHistories(false);
        Assert.assertTrue(createService(request).buildClientHistories(request, llmConfig, llmQuery));
    }

    /**
     * buildClientHistories：metadata __clientHistories=false 时覆盖 llmConfig
     */
    @Test
    public void testBuildClientHistories_fromMetadataFalse() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(ImmutableMap.of("__clientHistories", false));
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientHistories(true);
        Assert.assertFalse(createService(request).buildClientHistories(request, llmConfig, llmQuery));
    }

    /**
     * buildClientHistories：metadata 无 __clientHistories 时取 llmConfig 默认值
     */
    @Test
    public void testBuildClientHistories_defaultFromConfig() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(Message.build(ObjectBuilder.buildLLMQuery()));
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery(new HashMap<>());
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setClientHistories(true);
        Assert.assertTrue(createService(request).buildClientHistories(request, llmConfig, llmQuery));
        llmConfig.setClientHistories(false);
        Assert.assertFalse(createService(request).buildClientHistories(request, llmConfig, llmQuery));
    }

    private static final class DiscardHookService extends ProviderRequestService<OpenAiRequest> {
        @Override
        protected OpenAiRequest build() {
            throw new UnsupportedOperationException();
        }

        void runDiscard(OpenAiRequest request, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
            super.discard(request, llmConfig, llmQuery);
        }
    }
}
