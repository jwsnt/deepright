package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ShortcutNotifierTest {

    private ShortcutNotifier buildShortcutNotifier(WorkflowTask expectedTask, String expectedWorkflow, String expectedNotifier,
                                                   String expectedContent, Integer expectedCode, String expectedMetadataKey,
                                                   Object expectedMetadataValue) {
        return this.buildShortcutNotifier(expectedTask, expectedTask.getBiz(), expectedWorkflow, expectedNotifier, expectedContent, expectedCode, expectedMetadataKey, expectedMetadataValue);
    }

    private ShortcutNotifier buildShortcutNotifier(WorkflowTask expectedTask, String expectedBiz, String expectedWorkflow, String expectedNotifier,
                                                   String expectedContent, Integer expectedCode, String expectedMetadataKey,
                                                   Object expectedMetadataValue) {
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        shortcutNotifier.setNotifierService(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals(expectedContent, segment.getContent());
                Assert.assertEquals(expectedNotifier, segment.getNotifier());
                Assert.assertEquals(expectedBiz, segment.getBiz());
                Assert.assertEquals(expectedWorkflow, segment.getWorkflow());
                Assert.assertEquals(expectedCode, segment.getCode());
                Assert.assertEquals(expectedMetadataValue, segment.getMetadata().get(expectedMetadataKey));
                Assert.assertEquals(notifierWriteBack, expectedTask);
                Assert.assertEquals(redirectContext, expectedTask);
            }
        });
        return shortcutNotifier;
    }

    @Test
    public void testEndpoint1() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Notifier.ENDPOINT, segment.getNotifier());
                Assert.assertEquals(Integer.valueOf(250), segment.getCode());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.setTimeout4llm(1000);
        shortcutNotifier.endpoint(_workflowTask, metadata, "QUERY", 250);
        Assert.assertEquals(notifierManager, shortcutNotifier.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), shortcutNotifier.getTimeout4llm());
    }

    @Test
    public void testEndpoint2() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Notifier.ENDPOINT, segment.getNotifier());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.endpoint(_workflowTask, metadata, "QUERY");
    }

    @Test
    public void testEndpoint3() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Integer.valueOf(250), segment.getCode());
                Assert.assertEquals(Notifier.ENDPOINT, segment.getNotifier());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.endpoint(_workflowTask, "QUERY", 250);
    }

    @Test
    public void testEndpoint4() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Notifier.ENDPOINT, segment.getNotifier());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.endpoint(_workflowTask, "QUERY");
    }

    @Test
    public void testSource1() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Notifier.SOURCE, segment.getNotifier());
                Assert.assertEquals(Integer.valueOf(250), segment.getCode());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.source(_workflowTask, metadata, "QUERY", 250);
    }

    @Test
    public void testSource2() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Notifier.SOURCE, segment.getNotifier());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.source(_workflowTask, metadata, "QUERY");
    }

    @Test
    public void testSource3() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Integer.valueOf(250), segment.getCode());
                Assert.assertEquals(Notifier.SOURCE, segment.getNotifier());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.source(_workflowTask, "QUERY", 250);
    }

    @Test
    public void testSource4() throws Exception {
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals("QUERY", segment.getContent());
                Assert.assertEquals(Notifier.SOURCE, segment.getNotifier());
                Assert.assertEquals(notifierWriteBack, _workflowTask);
                Assert.assertEquals(redirectContext, _workflowTask);
            }
        };
        shortcutNotifier.setNotifierService(notifierManager);
        shortcutNotifier.source(_workflowTask, "QUERY");
    }

    @Test
    public void endpointAndSource_withNullContent_buildNullContentSegments() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.setQuery(null);
        ShortcutNotifier shortcutNotifier = new ShortcutNotifier();
        shortcutNotifier.setNotifierService(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) {
                Assert.assertNull(segment.getContent());
            }
        });

        shortcutNotifier.endpoint(workflowTask, null);
        shortcutNotifier.source(workflowTask, null);
    }

    @Test
    public void testEndpointWithWorkflowAndMetadataAndCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-x", Notifier.ENDPOINT, "QUERY", 250, "customKey", "customVal");

        shortcutNotifier.endpoint(workflowTask, "workflow-x", metadata, "QUERY", 250);
    }

    @Test
    public void testEndpointWithWorkflowAndMetadataUsesDefaultCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-y", Notifier.ENDPOINT, "QUERY", ProtocolCode.C200, "customKey", "customVal");

        shortcutNotifier.endpoint(workflowTask, "workflow-y", metadata, "QUERY");
    }

    @Test
    public void testEndpointWithWorkflowAndCodeUsesTaskMetadata() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-z", Notifier.ENDPOINT, "QUERY", 251, "taskKey", "taskVal");

        shortcutNotifier.endpoint(workflowTask, "workflow-z", "QUERY", 251);
    }

    @Test
    public void testEndpointWithWorkflowUsesTaskMetadataAndDefaultCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-a", Notifier.ENDPOINT, "QUERY", ProtocolCode.C200, "taskKey", "taskVal");

        shortcutNotifier.endpoint(workflowTask, "workflow-a", "QUERY");
    }

    @Test
    public void testEndpointWithBizWorkflowAndMetadataAndCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-explicit", "workflow-explicit", Notifier.ENDPOINT, "QUERY", 252, "customKey", "customVal");

        shortcutNotifier.endpoint(workflowTask, "biz-explicit", "workflow-explicit", metadata, "QUERY", 252);
    }

    @Test
    public void testEndpointWithBizWorkflowAndMetadataUsesSceneBiz() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-scene", "workflow-scene", Notifier.ENDPOINT, "QUERY", 254, "customKey", "customVal");

        shortcutNotifier.endpoint(workflowTask, "biz-fallback", "biz-scene@workflow-scene", metadata, "QUERY", 254);
    }

    @Test
    public void testEndpointWithBizWorkflowAndCodeUsesTaskMetadata() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-task", "workflow-task", Notifier.ENDPOINT, "QUERY", 253, "taskKey", "taskVal");

        shortcutNotifier.endpoint(workflowTask, "biz-task", "workflow-task", "QUERY", 253);
    }

    @Test
    public void testEndpointWithBizWorkflowUsesTaskMetadataAndCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-task-default", "workflow-task-default", Notifier.ENDPOINT, "QUERY", 255, "taskKey", "taskVal");

        shortcutNotifier.endpoint(workflowTask, "biz-task-default", "workflow-task-default", "QUERY", 255);
    }

    @Test
    public void testSourceWithWorkflowAndMetadataAndCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-b", Notifier.SOURCE, "QUERY", 350, "customKey", "customVal");

        shortcutNotifier.source(workflowTask, "workflow-b", metadata, "QUERY", 350);
    }

    @Test
    public void testSourceWithWorkflowAndMetadataUsesDefaultCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-c", Notifier.SOURCE, "QUERY", ProtocolCode.C200, "customKey", "customVal");

        shortcutNotifier.source(workflowTask, "workflow-c", metadata, "QUERY");
    }

    @Test
    public void testSourceWithWorkflowAndCodeUsesTaskMetadata() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-d", Notifier.SOURCE, "QUERY", 351, "taskKey", "taskVal");

        shortcutNotifier.source(workflowTask, "workflow-d", "QUERY", 351);
    }

    @Test
    public void testSourceWithWorkflowUsesTaskMetadataAndDefaultCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "workflow-e", Notifier.SOURCE, "QUERY", ProtocolCode.C200, "taskKey", "taskVal");

        shortcutNotifier.source(workflowTask, "workflow-e", "QUERY");
    }

    @Test
    public void testSourceWithBizWorkflowAndMetadataAndCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-source", "workflow-source", Notifier.SOURCE, "QUERY", 352, "customKey", "customVal");

        shortcutNotifier.source(workflowTask, "biz-source", "workflow-source", metadata, "QUERY", 352);
    }

    @Test
    public void testSourceWithBizWorkflowAndMetadataUsesSceneBizAndDefaultCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("customKey", "customVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-source-scene", "workflow-source-scene", Notifier.SOURCE, "QUERY", ProtocolCode.C200, "customKey", "customVal");

        shortcutNotifier.source(workflowTask, "biz-fallback", "biz-source-scene@workflow-source-scene", metadata, "QUERY");
    }

    @Test
    public void testSourceWithBizWorkflowAndCodeUsesTaskMetadata() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-source-task", "workflow-source-task", Notifier.SOURCE, "QUERY", 353, "taskKey", "taskVal");

        shortcutNotifier.source(workflowTask, "biz-source-task", "workflow-source-task", "QUERY", 353);
    }

    @Test
    public void testSourceWithBizWorkflowUsesTaskMetadataAndDefaultCode() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        workflowTask.putMetadata("taskKey", "taskVal");

        ShortcutNotifier shortcutNotifier = this.buildShortcutNotifier(
                workflowTask, "biz-source-task-default", "workflow-source-task-default", Notifier.SOURCE, "QUERY", ProtocolCode.C200, "taskKey", "taskVal");

        shortcutNotifier.source(workflowTask, "biz-source-task-default", "workflow-source-task-default", "QUERY");
    }
}
