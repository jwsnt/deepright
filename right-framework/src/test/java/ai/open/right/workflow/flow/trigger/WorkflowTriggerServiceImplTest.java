package ai.open.right.workflow.flow.trigger;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.trigger.impl.WorkflowTriggerServiceImpl;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class WorkflowTriggerServiceImplTest {

    @Test
    public void testInit() throws Exception {
        Map<String, WorkflowTrigger> triggerManagerMap = new HashMap<>();
        WorkflowTriggerServiceImpl.InitConfig service = new WorkflowTriggerServiceImpl.InitConfig();
        service.setTriggers(triggerManagerMap);
        WorkflowTriggerServiceImpl empty = (WorkflowTriggerServiceImpl) service.workflowTriggerService();
        Assert.assertEquals(triggerManagerMap, empty.getTriggers());
    }

    @Test
    public void testInit2() throws Exception {
        Map<String, WorkflowTrigger> triggerManagerMap = new HashMap<>();
        WorkflowTrigger workflowTrigger = new BaseTrigger();
        triggerManagerMap.put(WorkflowTrigger.NAME, workflowTrigger);
        WorkflowTriggerServiceImpl.InitConfig service = new WorkflowTriggerServiceImpl.InitConfig();
        service.setGlobal(workflowTrigger);
        service.setTriggers(triggerManagerMap);
        WorkflowTriggerServiceImpl empty = (WorkflowTriggerServiceImpl) service.workflowTriggerService();
        Assert.assertTrue(empty.getTriggers().isEmpty());
    }

    @Test
    public void testGlobal() throws Exception {
        WorkflowTriggerServiceImpl service = new WorkflowTriggerServiceImpl();
        WorkflowConfig _workflowConfig = new WorkflowConfig();
        WorkflowTask _workflowTask = ObjectBuilder.buildWorkflowTask();
        AtomicInteger integer = new AtomicInteger();
        service.setGlobal(new WorkflowTrigger() {
            @Override
            public void before(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
                Assert.assertEquals(workflowConfig, _workflowConfig);
                Assert.assertEquals(workTask, _workflowTask);
                integer.incrementAndGet();
            }
        });
        service.before(_workflowConfig, _workflowTask);
        Assert.assertEquals(1, integer.get());
    }
}
