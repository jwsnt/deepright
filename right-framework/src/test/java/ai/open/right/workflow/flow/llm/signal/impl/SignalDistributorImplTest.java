package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class SignalDistributorImplTest {

    @Test
    public void testGeSignalResponse() throws Exception {
        WorkflowTask workflowTask = ObjectBuilder.buildWorkflowTask();
        SyncWorkflowTask syncWorkflowTask = new SyncWorkflowTask(workflowTask, null, 1000) {
            public String get() {
                return "OK";
            }
        };

        SignalDistributorImpl distributor = new SignalDistributorImpl();
        Assert.assertEquals("OK", distributor.getSignalResponse(Arrays.asList(syncWorkflowTask)));
    }

    @Test
    public void notify_buildsSegmentForNonNullResponse() throws Exception {
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        SignalDistributorImpl distributor = new SignalDistributorImpl();
        distributor.setNotifierService(new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, ai.open.right.context.RedirectContext redirectContext,
                               ai.open.right.workflow.notify.NotifierWriteBack notifierWriteBack) {
                Assert.assertEquals("response", segment.getContent());
            }
        });

        distributor.notify(message, "response", "TARGET");
    }

    @Test(expected = IllegalArgumentException.class)
    public void notify_rejectsNullResponseBeforeBuildingSegment() throws Exception {
        SignalDistributorImpl distributor = new SignalDistributorImpl();
        distributor.notify(Message.build(ObjectBuilder.buildLLMQuery()), null, "TARGET");
    }

    @Test
    public void testDistribute() throws Exception {
        Map<String, String> workflows = new HashMap<String, String>();
        workflows.put("KEY1", "VAL1");
        SignalConfig config = new SignalConfig();
        config.setConfigs(workflows);
        SignalDistributorImpl distributor = new SignalDistributorImpl();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        distributor.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        distributor.distribute(config, "KEY1=B", message);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testDistributeWithOutKey() throws Exception {
        Map<String, String> workflows = new HashMap<String, String>();
        workflows.put("KEY_1", "VAL1");
        SignalConfig config = new SignalConfig();
        config.setConfigs(workflows);
        SignalDistributorImpl distributor = new SignalDistributorImpl();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        distributor.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        distributor.distribute(config, "KEY1=B", message);
    }

    @Test
    public void testDistributeForMulti() throws Exception {
        Map<String, String> workflows = new HashMap<String, String>();
        workflows.put("KEY1", "VAL1");
        SignalConfig config = new SignalConfig();
        config.setSynthesizer("NextWorkflow");
        config.setConfigs(workflows);
        SignalDistributorImpl distributor = new SignalDistributorImpl() {
            protected List<SyncWorkflowTask> getSyncWorkflowTasks(SignalConfig signalConfig, List<String> signal, Message message) throws Exception {
                return null;
            }

            protected String getSignalResponse(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
                return "OK";
            }
        };
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        distributor.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        distributor.distribute(config, Arrays.asList("KEY1=B", "B=C"), message);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testDistributeForMultiWithOutKey() throws Exception {
        Map<String, String> workflows = new HashMap<String, String>();
        workflows.put("KEY1", "VAL1");
        SignalConfig config = new SignalConfig();
        // Test Empty
        config.setSynthesizer(null);
        config.setConfigs(workflows);
        SignalDistributorImpl distributor = new SignalDistributorImpl() {
            protected List<SyncWorkflowTask> getSyncWorkflowTasks(SignalConfig signalConfig, List<String> signal, Message message) throws Exception {
                return null;
            }

            protected String getSignalResponse(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
                return "OK";
            }
        };
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        distributor.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        distributor.distribute(config, Arrays.asList("KEY1=B", "B=C"), message);
    }


    @Test
    public void testGetSyncWorkflowTasks() throws Exception {
        Map<String, String> workflows = new HashMap<String, String>();
        workflows.put("KEY1", "VAL1");
        SignalConfig config = new SignalConfig();
        config.setConfigs(workflows);
        SignalDistributorImpl distributor = new SignalDistributorImpl();
        Message message = Message.build(ObjectBuilder.buildLLMQuery());
        distributor.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        distributor.getSyncWorkflowTasks(config, Arrays.asList("KEY1=B"), message);
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        SignalDistributorImpl.InitConfig service = new SignalDistributorImpl.InitConfig();
        service.setNotifierService(notifierManager);
        service.setTimeout4Llm(1000);
        SignalDistributorImpl empty = (SignalDistributorImpl) service.signalDistributor();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(Integer.valueOf(1000), empty.getTimeout4Llm());
    }
}
