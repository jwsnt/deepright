package ai.open.right.workflow.flow.assistant;

import ai.open.right.ObjectBuilder;
import ai.open.right.protocol.Protocol;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ChainAssistantTest {

    @Test
    public void testSetGet() throws Exception {
        ChainAssistant chainAssistant = new ChainAssistant() {
        };
        chainAssistant.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        Assert.assertNotNull(chainAssistant.getNotifierService());
    }

    @Test
    public void notify_withNullContent_buildsNullContentSegment() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(null);
        ChainAssistant assistant = new ChainAssistant() {
        };
        assistant.setNotifierService(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, ai.open.right.context.RedirectContext redirectContext,
                               NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) {
                Assert.assertNull(segment.getContent());
            }
        });

        assistant.notify(workflowTask, workflowTask.getWorkflow(), null, null,
                Notifier.LOCALHOST, Protocol.CHAT, null, ProtocolCode.C200);
    }

    @Test
    public void testChain1() throws Exception {
        Map<String, Object> _metadata = new HashMap<String, Object>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(metadata, _metadata);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _metadata, _content);
    }

    @Test
    public void testChain2() throws Exception {
        Map<String, Object> _metadata = new HashMap<String, Object>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        Integer _code = 123;
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(metadata, _metadata);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
                Assert.assertEquals(code, _code);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _metadata, _content, _code);
    }

    @Test
    public void testChain3() throws Exception {
        List<MediaContext> _mediaContext = new ArrayList<MediaContext>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(mediaContext, _mediaContext);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _mediaContext, _content);
    }

    @Test
    public void testChain4() throws Exception {
        List<MediaContext> _mediaContext = new ArrayList<MediaContext>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        Integer _code = 123;
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(mediaContext, _mediaContext);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
                Assert.assertEquals(code, _code);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _mediaContext, _content, _code);
    }

    @Test
    public void testChain5() throws Exception {
        List<MediaContext> _mediaContext = new ArrayList<MediaContext>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        String _protocol = "BCD";
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(mediaContext, _mediaContext);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
                Assert.assertEquals(protocol, _protocol);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _mediaContext, _protocol, _content);
    }

    @Test
    public void testChain6() throws Exception {
        List<MediaContext> _mediaContext = new ArrayList<MediaContext>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        String _protocol = "BCD";
        Integer _code = 123;
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(mediaContext, _mediaContext);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
                Assert.assertEquals(protocol, _protocol);
                Assert.assertEquals(code, _code);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _mediaContext, _protocol, _content, _code);
    }

    @Test
    public void testChain7() throws Exception {
        Map<String, Object> _metadata = new HashMap<String, Object>();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        String _content = "ABC";
        String _protocol = "BCD";
        Integer _code = 123;
        ChainAssistant chainAssistant = new ChainAssistant() {
            public void chainOr2Endpoint(WorkflowConfig workflowConfig, WorkflowTask workTask, Map<String, Object> metadata, List<MediaContext> mediaContext, String protocol, String content, Integer code) throws Exception {
                Assert.assertEquals(metadata, _metadata);
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                Assert.assertEquals(content, _content);
                Assert.assertEquals(protocol, _protocol);
                Assert.assertEquals(code, _code);
            }
        };
        chainAssistant.chainOr2Endpoint(_workflowConfig, _workflowTask, _metadata, _protocol, _content, _code);
    }

    @Test
    public void testHashCode() throws Exception {
        Object object = ChainAssistant.ChainInitConfig.class.getConstructor(null).newInstance(null);
        Assert.assertEquals(object.hashCode(), object.hashCode());
        Assert.assertEquals(object, object);
    }

    /**
     * notify(workTask, metadata, notifier, content) 使用 workTask.getWorkflow() 作为 workflow，
     * 并透传 metadata，media 为 null，protocol=CHAT，code=C200
     */
    @Test
    public void notify_withMetadata_usesWorkTaskWorkflowAndDelegatesToFullNotify() throws Exception {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        wt.setWorkflow("WF_FROM_TASK");
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put("source", "unit-test");
        NotifyCaptureAssistant assistant = new NotifyCaptureAssistant();
        assistant.notify(wt, metadata, "custom-notifier", "body");
        Assert.assertSame(wt, assistant.lastTask);
        Assert.assertEquals("WF_FROM_TASK", assistant.lastWorkflow);
        Assert.assertSame(metadata, assistant.lastMetadata);
        Assert.assertNull(assistant.lastMedia);
        Assert.assertEquals("custom-notifier", assistant.lastNotifier);
        Assert.assertEquals(Protocol.CHAT, assistant.lastProtocol);
        Assert.assertEquals("body", assistant.lastContent);
        Assert.assertEquals(ProtocolCode.C200, assistant.lastCode);
    }

    @Test
    public void notify_withWorkflowAndMetadata_delegatesToFullNotify() throws Exception {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<String, Object>();
        metadata.put("source", "unit-test");
        NotifyCaptureAssistant assistant = new NotifyCaptureAssistant();
        assistant.notify(wt, "WF_CHAIN", metadata, "custom-notifier", "body");
        Assert.assertSame(wt, assistant.lastTask);
        Assert.assertEquals("WF_CHAIN", assistant.lastWorkflow);
        Assert.assertSame(metadata, assistant.lastMetadata);
        Assert.assertNull(assistant.lastMedia);
        Assert.assertEquals("custom-notifier", assistant.lastNotifier);
        Assert.assertEquals(Protocol.CHAT, assistant.lastProtocol);
        Assert.assertEquals("body", assistant.lastContent);
        Assert.assertEquals(ProtocolCode.C200, assistant.lastCode);
    }

    /**
     * notify(workTask, workflow, notifier, content) 委托到完整 notify：metadata/media 为 null，protocol=CHAT，code=C200
     */
    @Test
    public void notify_withNotifier_delegatesToFullNotify() throws Exception {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        NotifyCaptureAssistant assistant = new NotifyCaptureAssistant();
        assistant.notify(wt, "WF_CHAIN", "custom-notifier", "body");
        Assert.assertSame(wt, assistant.lastTask);
        Assert.assertEquals("WF_CHAIN", assistant.lastWorkflow);
        Assert.assertNull(assistant.lastMetadata);
        Assert.assertNull(assistant.lastMedia);
        Assert.assertEquals("custom-notifier", assistant.lastNotifier);
        Assert.assertEquals(Protocol.CHAT, assistant.lastProtocol);
        Assert.assertEquals("body", assistant.lastContent);
        Assert.assertEquals(ProtocolCode.C200, assistant.lastCode);
    }

    /**
     * notify(workTask, notifier, content) 使用 workTask.getWorkflow() 作为 workflow 委托到完整 notify
     */
    @Test
    public void notify_threeArg_usesWorkTaskWorkflow() throws Exception {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        wt.setWorkflow("WF_FROM_TASK");
        NotifyCaptureAssistant assistant = new NotifyCaptureAssistant();
        assistant.notify(wt, "custom-notifier", "body");
        Assert.assertSame(wt, assistant.lastTask);
        Assert.assertEquals("WF_FROM_TASK", assistant.lastWorkflow);
        Assert.assertNull(assistant.lastMetadata);
        Assert.assertNull(assistant.lastMedia);
        Assert.assertEquals("custom-notifier", assistant.lastNotifier);
        Assert.assertEquals(Protocol.CHAT, assistant.lastProtocol);
        Assert.assertEquals("body", assistant.lastContent);
        Assert.assertEquals(ProtocolCode.C200, assistant.lastCode);
    }

    /**
     * notify(workTask, workflow, notifier, content)：显式 ENDPOINT + 指定 workflow 时与完整 notify 一致（原两参数 workflow+content 便捷方法已移除）
     */
    @Test
    public void notify_fourArg_withEndpointNotifier_delegatesToFullNotify() throws Exception {
        WorkflowTask wt = ObjectBuilder.buildWorkflowTask();
        NotifyCaptureAssistant assistant = new NotifyCaptureAssistant();
        assistant.notify(wt, "WF_ONLY", Notifier.ENDPOINT, "payload");
        Assert.assertSame(wt, assistant.lastTask);
        Assert.assertEquals("WF_ONLY", assistant.lastWorkflow);
        Assert.assertNull(assistant.lastMetadata);
        Assert.assertNull(assistant.lastMedia);
        Assert.assertEquals(Notifier.ENDPOINT, assistant.lastNotifier);
        Assert.assertEquals(Protocol.CHAT, assistant.lastProtocol);
        Assert.assertEquals("payload", assistant.lastContent);
        Assert.assertEquals(ProtocolCode.C200, assistant.lastCode);
    }

    private static final class NotifyCaptureAssistant extends ChainAssistant {
        WorkflowTask lastTask;
        String lastWorkflow;
        Map<String, Object> lastMetadata;
        List<MediaContext> lastMedia;
        String lastNotifier;
        String lastProtocol;
        String lastContent;
        Integer lastCode;

        @Override
        protected void notify(WorkflowTask workTask, String workflow, Map<String, Object> metadata, List<MediaContext> mediaContext, String notifier, String protocol, String content, Integer code) throws Exception {
            this.lastTask = workTask;
            this.lastWorkflow = workflow;
            this.lastMetadata = metadata;
            this.lastMedia = mediaContext;
            this.lastNotifier = notifier;
            this.lastProtocol = protocol;
            this.lastContent = content;
            this.lastCode = code;
        }
    }
}
