package ai.deepright.memory.impl;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.feature.FeatureUtils;
import ai.deepright.memory.MemoryRecall;
import ai.deepright.memory.MemoryService;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;

import java.util.ArrayList;
import java.util.List;

@Slf4j
public class MultiMemoryService implements MemoryService {

    protected List<MemoryService> services = new ArrayList<MemoryService>();

    public MultiMemoryService add(MemoryService service) throws Exception {
        this.services.add(service);
        return this;
    }

    @Override
    public String init(WorkflowTask workTask) throws Exception {
        StringBuffer buffer = new StringBuffer();
        for (MemoryService service : this.services) {
            try {
                buffer.append(service.init(workTask)).append(FeatureUtils.buildLineSeparator(workTask));
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
        return buffer.toString();
    }

    @Override
    public String recall(WorkflowTask workTask, MemoryRecall recall) throws Exception {
        StringBuffer buffer = new StringBuffer();
        for (MemoryService service : this.services) {
            try {
                buffer.append(service.recall(workTask, recall)).append(FeatureUtils.buildLineSeparator(workTask));
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
        return buffer.toString();
    }

    @Override
    public void commit(WorkflowTask workTask) throws Exception {
        for (MemoryService service : this.services) {
            try {
                service.commit(workTask);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    @Override
    public void refresh(WorkflowTask workTask, List<History> histories) throws Exception {
        for (MemoryService service : this.services) {
            try {
                service.refresh(workTask, histories);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    @Override
    public Boolean support(WorkflowTask workTask) throws Exception {
        return true;
    }

    public Boolean isEmpty() throws Exception {
        return CollectionUtils.isEmpty(this.services);
    }
}
