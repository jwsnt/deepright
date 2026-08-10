package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class DefaultAssistantTest {

    @Test
    public void testConfig() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        // Without Exception
        defAssistant.config(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask());
    }

    @Test
    public void testConfigFunCall() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmFunCall(new LLMFunCall());
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMFunCallRequest request = ProviderFunCallRequest.builder()
                .name("HELLO")
                .build();
        workflowTask.getMetadata().put(ProviderRequestService.KEY_FUN_FETCH, request);
        workflowTask.setQuery(JsonUtils.write(ImmutableMap.of(WorkflowConfig.UNBOXED, ImmutableMap.of("A", "B"))));
        defAssistant.config(workflowConfig, workflowTask);
        Assert.assertEquals("{\"A\":\"B\"}", workflowTask.getQuery());
    }

    @Test
    public void testJSONModeWithUnbox() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setRequest(WorkflowConfig.REQUEST_JSON);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.write(ImmutableMap.of("__query", ImmutableMap.of("A", "B"))));
        defAssistant.config(workflowConfig, workflowTask);
        Assert.assertEquals("{\"A\":\"B\"}", workflowTask.getQuery());
    }

    @Test
    public void testConfigFunCallWithUnbox() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmFunCall(new LLMFunCall());
        workflowConfig.setUnboxed("_box");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        LLMFunCallRequest request = ProviderFunCallRequest.builder()
                .name("HELLO")
                .build();
        workflowTask.getMetadata().put(ProviderRequestService.KEY_FUN_FETCH, request);
        workflowTask.setQuery(JsonUtils.write(ImmutableMap.of("_box", ImmutableMap.of("A", "B"))));
        defAssistant.config(workflowConfig, workflowTask);
        Assert.assertEquals("{\"A\":\"B\"}", workflowTask.getQuery());
    }

    @Test
    public void testConfigFunCallWithNull() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmFunCall(new LLMFunCall());
        workflowConfig.setUnboxed("_box");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.getMetadata().put(ProviderRequestService.KEY_FUN_FETCH, "HELLO");
        workflowTask.setQuery(JsonUtils.write(ImmutableMap.of("A", "B")));
        defAssistant.config(workflowConfig, workflowTask);
        Assert.assertEquals("{\"A\":\"B\"}", workflowTask.getQuery());
    }

    @Test
    public void testConfigFunCallWithNotUnBox() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmFunCall(new LLMFunCall());
        workflowConfig.setRequest(WorkflowConfig.REQUEST_JSON);
        workflowConfig.setUnboxed("_box");
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(JsonUtils.write(ImmutableMap.of("A", "B")));
        defAssistant.config(workflowConfig, workflowTask);
        Assert.assertEquals("{\"A\":\"B\"}", workflowTask.getQuery());
    }

    @Test
    public void testConfigFunCallWithException() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setLlmFunCall(new LLMFunCall());
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.getMetadata().put(ProviderRequestService.KEY_FUN_FETCH, "HELLO");
        workflowTask.setQuery("[1,2,3]");
        defAssistant.config(workflowConfig, workflowTask);
        Assert.assertEquals("[1,2,3]", workflowTask.getQuery());
    }

    @Test
    public void testAllowed() {
        DefaultAssistant defAssistant = new DefaultAssistant();
        Assert.assertTrue(defAssistant.allowed(ObjectBuilder.buildWorkflowTask()));
    }

    @Test(expected = IllegalArgumentException.class)
    public void testExecuteWithErrorProtocol() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setProtocol(Protocol.TOOL);
        defAssistant.execute(new WorkflowConfig(), workflowTask);
    }

    @Test
    public void testExecuteWithNotAllowed() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant() {
            public Boolean allowed(WorkflowTask workflowTask) {
                return false;
            }
        };
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        defAssistant.execute(new WorkflowConfig(), workflowTask);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testExecuteWithNotLlmConfig() throws Exception {
        DefaultAssistant defAssistant = new DefaultAssistant();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setProtocol("PROTOCOL");
        defAssistant.execute(new WorkflowConfig(), workflowTask);
    }

    @Test
    public void testExecuteWithRun() throws Exception {
        LLMQueryService llmQueryService = EasyMock.createMock(LLMQueryService.class);
        llmQueryService.query(EasyMock.anyObject(LLMQuery.class), EasyMock.anyObject(LLMConfig.class), EasyMock.anyObject(SignalStream.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(llmQueryService);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<String, LLMQueryService>();
        llmQueryServices.put(LLMQueryService.LLM_COZE, llmQueryService);
        DefaultAssistant defAssistant = new DefaultAssistant();
        defAssistant.setLlmQueryService(llmQueryServices);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        LLMConfig llmConfig = new LLMConfig();
        llmConfig.setProvider(LLMQueryService.LLM_COZE);
        workflowConfig.setLlmConfig(llmConfig);
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        EasyMock.expect(signalFactory.signal(workflowConfig)).andReturn(signalStream).anyTimes();
        EasyMock.replay(signalFactory, signalStream);
        defAssistant.setSignalFactory(signalFactory);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        defAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(llmQueryService, signalFactory);
    }

    @Test
    public void testChain() throws Exception {
        AtomicInteger count = new AtomicInteger();
        DefaultAssistant defAssistant = new DefaultAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                count.getAndIncrement();
            }
        };
        WorkflowConfig config = new WorkflowConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        defAssistant.chainOr2Endpoint(config, workflowTask, Protocol.CHAT, "HELLO", ProtocolCode.C500);
        defAssistant.chainOr2Endpoint(config, workflowTask, "HELLO", ProtocolCode.C500);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(count.get()));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        EasyMock.replay(notifierManager, signalFactory);
        DefaultAssistant.InitConfig assistant = new DefaultAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        DefaultAssistant empty = assistant.defaultAssistant();
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(notifierManager, signalFactory);
    }
}
