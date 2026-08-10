package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.context.RedirectContext;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.command.QuickCommand;
import ai.open.right.workflow.flow.command.QuickCommandStore;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalFactory;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.tools.ToolsBody;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import ai.open.right.workflow.flow.tools.ToolsResponse;
import ai.open.right.workflow.flow.tools.ToolsService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ToolsAssistantTest {

    @Test
    public void testStoreWithoutQuickCommand() {
        QuickCommandStore quickCommandStore = EasyMock.createMock(QuickCommandStore.class);
        quickCommandStore.store(EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyString(), EasyMock.anyString(), EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        toolsAssistant.setQuickCommandStore(quickCommandStore);
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setData(new ToolsBody());
        toolsAssistant.storeQuickCommand(ObjectBuilder.buildWorkflowTask(), new ToolsConfig(), toolsResponse);
    }

    @Test
    public void testStoreWithQuickCommand() {
        QuickCommandStore quickCommandStore = EasyMock.createMock(QuickCommandStore.class);
        quickCommandStore.store(EasyMock.anyObject(List.class), EasyMock.anyInt(), EasyMock.anyString(), EasyMock.anyString(), EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        toolsAssistant.setQuickCommandStore(quickCommandStore);
        ToolsResponse toolsResponse = new ToolsResponse();
        ToolsBody toolsBody = new ToolsBody();
        QuickCommand quickCommand = new QuickCommand();
        quickCommand.setCommand("Command");
        quickCommand.setContent("Content");
        quickCommand.setPriority(100L);
        toolsBody.setCommand(Arrays.asList(quickCommand));
        toolsResponse.setData(toolsBody);
        toolsAssistant.storeQuickCommand(ObjectBuilder.buildWorkflowTask(), new ToolsConfig(), toolsResponse);
    }

    @Test(expected = WorkflowException.class)
    public void testDoingToolsResponseWithFailedAndHasMessage() throws Exception {
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C500);
        toolsResponse.setMsg("Failed");
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        toolsAssistant.setNotifierService(notifierManager);
        toolsAssistant.doingToolsResponse(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test(expected = WorkflowException.class)
    public void testDoingToolsResponseWithFailedAndNotMessage() throws Exception {
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setData(new ToolsBody());
        toolsResponse.setCode(ProtocolCode.C500);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        toolsAssistant.setNotifierService(notifierManager);
        toolsAssistant.doingToolsResponse(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testDoingToolsResponseWithParseFailed() throws Exception {
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        toolsAssistant.setNotifierService(ObjectBuilder.buildAssertNotifierManagerWithOnlyAssert("{\"code\":200,\"data\":[{\"K1\":\"V1\"}]}"));
        toolsAssistant.doingToolsResponse(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask(), "{\"code\":200,\"data\":[{\"K1\":\"V1\"}]}");
    }

    @Test
    public void testDoingToolsResponseWithFailedAndNotMessageWithSuccessCode() throws Exception {
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setData(new ToolsBody());
        toolsResponse.setCode(ProtocolCode.C500);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        toolsAssistant.setNotifierService(notifierManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ToolsConfig toolsConfig = new ToolsConfig();
        workflowConfig.setToolsConfig(toolsConfig);
        toolsConfig.setSuccessCode(500);
        toolsAssistant.doingToolsResponse(workflowConfig, ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testDoingToolsResponseWithSuccessAndNotData() throws Exception {
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C200);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals("{\"code\":200}", segment.getContent());
            }
        };
        toolsAssistant.setNotifierService(notifierManager);
        toolsAssistant.doingToolsResponse(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testDoingToolsResponseWithSuccessAndHasDataWithoutChain() throws Exception {
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C200);
        ToolsBody toolsBody = new ToolsBody();
        toolsBody.setContent("HELLO");
        toolsResponse.setData(toolsBody);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        toolsAssistant.setNotifierService(notifierManager);
        toolsAssistant.doingToolsResponse(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testDoingToolsResponseWithSuccessAndHasDataWithChain() throws Exception {
        ToolsBody toolsBody = new ToolsBody();
        toolsBody.setContent("HELLO WORLD");
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C200);
        toolsResponse.setData(toolsBody);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        toolsAssistant.setNotifierService(notifierManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("NextWorkflow");
        toolsAssistant.doingToolsResponse(workflowConfig, ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testDoingToolsResponseWithEmptyResponse() throws Exception {
        ToolsBody toolsBody = new ToolsBody();
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C200);
        toolsResponse.setData(toolsBody);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        toolsAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("NextWorkflow");
        toolsAssistant.doingToolsResponse(workflowConfig, ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test(expected = WorkflowException.class)
    public void testDoingToolsResponseWithEmptyResponseAndSuccessCode() throws Exception {
        ToolsBody toolsBody = new ToolsBody();
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C200);
        toolsResponse.setData(toolsBody);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("UNKNOWN", segment.getContent());
            }
        };
        toolsAssistant.setNotifierService(notifierManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setChain("NextWorkflow");
        ToolsConfig toolsConfig = new ToolsConfig();
        workflowConfig.setToolsConfig(toolsConfig);
        toolsConfig.setSuccessCode(500);
        toolsAssistant.doingToolsResponse(workflowConfig, ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testExecutor() throws Exception {
        ToolsAssistant toolsAssistant = new ToolsAssistant() {
            @Override
            public void doingToolsResponse(WorkflowConfig workConfig, WorkflowTask workTask, String response) throws Exception {

            }
        };
        ToolsService toolsService = EasyMock.createMock(ToolsService.class);
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        workflowConfig.setToolsConfig(new ToolsConfig());
        EasyMock.expect(toolsService.execute(workflowConfig.getToolsConfig(), workflowTask)).andReturn("{\"code\":200}").anyTimes();
        EasyMock.replay(toolsService);
        toolsAssistant.setToolsService(toolsService);
        toolsAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackDirect());
        toolsAssistant.execute(workflowConfig, workflowTask);
        EasyMock.verify(toolsService);
    }

    @Test
    public void testDoingToolsResponseWithNotConvert() throws Exception {
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals("{\"KEY\":\"HELLO\"}", segment.getContent());
            }
        };
        toolsAssistant.setNotifierService(notifierManager);
        toolsAssistant.doingToolsResponse(new WorkflowConfig(), ObjectBuilder.buildWorkflowTask(), "{\"KEY\":\"HELLO\"}");
    }

    @Test
    public void testDoingToolsResponseWithJsonParseException() throws Exception {
        final String[] result = new String[2];
        ToolsAssistant toolsAssistant = new ToolsAssistant() {
            @Override
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, String protocol, String content) throws Exception {
                result[0] = protocol;
                result[1] = content;
            }
        };
        WorkflowConfig workflowConfig = new WorkflowConfig();
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        String response = "INVALID_JSON";
        toolsAssistant.doingToolsResponse(workflowConfig, workflowTask, response);
        Assert.assertEquals(Protocol.TOOL, result[0]);
        Assert.assertEquals(response, result[1]);
    }

    @Test
    public void testDoingToolsResponseWithSourceQuery() throws Exception {
        ToolsBody toolsBody = new ToolsBody();
        toolsBody.setContent("HELLO WORLD");
        ToolsResponse toolsResponse = new ToolsResponse();
        toolsResponse.setCode(ProtocolCode.C200);
        toolsResponse.setData(toolsBody);
        ToolsAssistant toolsAssistant = new ToolsAssistant();
        NotifierServiceImpl notifierManager = ObjectBuilder.buildNotifierManagerWithimplement();
        toolsAssistant.setNotifierService(notifierManager);
        WorkflowConfig workflowConfig = new WorkflowConfig();
        ToolsConfig toolsConfig = new ToolsConfig();
        toolsConfig.setSource(true);
        workflowConfig.setToolsConfig(toolsConfig);
        workflowConfig.setChain("NextWorkflow");
        toolsAssistant.doingToolsResponse(workflowConfig, ObjectBuilder.buildWorkflowTask(), JsonUtils.write(toolsResponse));
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = EasyMock.createMock(NotifierServiceImpl.class);
        SignalFactory signalFactory = EasyMock.createMock(SignalFactory.class);
        Map<String, LLMQueryService> llmQueryServices = new HashMap<>();
        WorkflowConfig workflowConfig = new WorkflowConfig();
        MediaConfig mediaConfig = new MediaConfig();
        workflowConfig.setMediaConfig(mediaConfig);
        QuickCommandStore service1 = EasyMock.createMock(QuickCommandStore.class);
        ToolsService service2 = EasyMock.createMock(ToolsService.class);
        EasyMock.replay(service1, service2, notifierManager, signalFactory);
        ToolsAssistant.InitConfig assistant = new ToolsAssistant.InitConfig();
        assistant.setNotifierService(notifierManager);
        assistant.setLlmQueryService(llmQueryServices);
        assistant.setSignalFactory(signalFactory);
        assistant.setQuickCommandStore(service1);
        assistant.setToolsService(service2);
        ToolsAssistant empty = assistant.toolsAssistant();
        Assert.assertEquals(service1, empty.getQuickCommandStore());
        Assert.assertEquals(service2, empty.getToolsService());
        Assert.assertEquals(empty.getLlmQueryService(), llmQueryServices);
        Assert.assertEquals(empty.getNotifierService(), notifierManager);
        Assert.assertEquals(empty.getSignalFactory(), signalFactory);
        EasyMock.verify(service1, service2, notifierManager, signalFactory);
    }

    @Test
    public void testHashCode1() throws Exception {
        Object object = ToolsAssistant.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    @Test
    public void testHashCode2() throws Exception {
        Object object = ToolsAssistant.InitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }
}
